//go:generate go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen@v1.12.4 --config=types.cfg.yaml ../apispec/api.spec.json
//go:generate go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen@v1.12.4 --config=server.cfg.yaml ../apispec/api.spec.json

package controllers

import (
	"encoding/json"
	"fmt"
	"github.com/RedHatInsights/entitlements-api-go/logger"
	"github.com/sirupsen/logrus"
	"net/http"

	"github.com/RedHatInsights/entitlements-api-go/ams"
	"github.com/RedHatInsights/entitlements-api-go/api"
	"github.com/RedHatInsights/entitlements-api-go/bop"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

type SeatManagerApi struct {
	client ams.AMSInterface
	bop    bop.Bop
}

const BASE_LINK_URL = "/api/entitlements/v1/seats"

var _ api.ServerInterface = &SeatManagerApi{}

func NewSeatManagerApi(cl ams.AMSInterface, bopClient bop.Bop) *SeatManagerApi {
	return &SeatManagerApi{
		client: cl,
		bop:    bopClient,
	}
}

func doError(w http.ResponseWriter, code int, err error) {
	response := api.Error{
		Error: toPtr(err.Error()),
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(response)
}

func do500(w http.ResponseWriter, err error) {
	logger.Log.WithFields(logrus.Fields{"error": err}).Error(err)
	doError(w, http.StatusInternalServerError, err)
}

func (s *SeatManagerApi) DeleteSeatsId(w http.ResponseWriter, r *http.Request, id string) {
	idObj := identity.Get(r.Context()).Identity

	if !idObj.User.OrgAdmin {
		doError(w, http.StatusForbidden, fmt.Errorf("Not allowed to delete subscription %s", id))
		return
	}

	subscription, err := s.client.GetSubscription(id)
	if err != nil {
		do500(w, err)
		return
	}
	orgId, ok := subscription.GetOrganizationID()
	if !ok {
		do500(w, fmt.Errorf("subscription %s does not have an organization", id))
		return
	}

	if orgId != idObj.Internal.OrgID {
		doError(w, http.StatusForbidden, fmt.Errorf("Not allowed to delete subscription %s", id))
	}

	if err = s.client.DeleteSubscription(id); err != nil {
		do500(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toPtr[T any](s T) *T {
	return &s
}

func fillDefaults(params *api.GetSeatsParams) {
	if params.Limit == nil {
		params.Limit = toPtr(10)
	}

	if params.Offset == nil {
		params.Offset = toPtr(0)
	}
}

func (s *SeatManagerApi) GetSeats(w http.ResponseWriter, r *http.Request, params api.GetSeatsParams) {

	idObj := identity.Get(r.Context()).Identity

	// AMS uses fixed pages rather than offsets So we are forcing the
	// offset to be tied to the nearest previous page.
	fillDefaults(&params)
	limit := int(*params.Limit)
	offset := int(*params.Offset)

	if limit < 1 {
		doError(w, http.StatusBadRequest, fmt.Errorf("limit must be > 0"))
		return
	}

	if offset < 0 {
		doError(w, http.StatusBadRequest, fmt.Errorf("offset must be >= 0"))
		return
	}

	page := 1 + (offset / limit)

	subs, err := s.client.GetSubscriptions(idObj.Internal.OrgID, limit, page)
	if err != nil {
		do500(w, err)
		return
	}

	quotaCost, err := s.client.GetQuotaCost(idObj.Internal.OrgID)
	if err != nil {
		do500(w, err)
		return
	}

	var seats []api.Seat
	for _, sub := range subs.Slice() {
		seats = append(seats, api.Seat{
			AccountUsername: toPtr(sub.Creator().Username()),
			SubscriptionId:  toPtr(sub.ID()),
		})
	}

	// AMS api for subscriptions doesn't return the information needed to calculate last
	prev_offset := offset - limit
	if prev_offset < 0 {
		prev_offset = 0
	}

	links := &api.PaginationLinks{
		First:    toPtr(fmt.Sprintf("%s/?limit=%d&offset=0", BASE_LINK_URL, limit)),
		Next:     toPtr(fmt.Sprintf("%s/?limit=%d&offset=%d", BASE_LINK_URL, limit, offset+limit)),
		Previous: toPtr(fmt.Sprintf("%s/?limit=%d&offset=%d", BASE_LINK_URL, limit, prev_offset)),
	}

	resp := api.ListSeatsResponsePagination{
		Meta: &api.PaginationMeta{
			Count: toPtr(int64(len(seats))),
		},
		Links:    links,
		Data:     seats,
		Allowed:  toPtr(int64(quotaCost.Allowed())),
		Consumed: toPtr(int64(quotaCost.Consumed())),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(resp); err != nil {
		do500(w, err)
		return
	}

}

func (s *SeatManagerApi) PostSeats(w http.ResponseWriter, r *http.Request) {
	idObj := identity.Get(r.Context()).Identity

	if !idObj.User.OrgAdmin {
		doError(w, http.StatusForbidden, fmt.Errorf("Not allowed to assign seats"))
		return
	}

	seat := new(api.SeatRequest)
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(seat); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	user, err := s.bop.GetUser(seat.AccountUsername)
	if err != nil {
		do500(w, fmt.Errorf("BOP GetUser [%w]", err))
		return
	}

	if user.OrgId != idObj.Internal.OrgID {
		doError(w, http.StatusForbidden, fmt.Errorf("Not allowed to assign seats to users outside of Organization %s", idObj.Internal.OrgID))
	}

	quotaCost, err := s.client.GetQuotaCost(idObj.Internal.OrgID)
	if err != nil {

		do500(w, fmt.Errorf("GetQuotaCost [%w]", err))
		return
	}

	resp, err := s.client.QuotaAuthorization(seat.AccountUsername, quotaCost.Version())
	if err != nil {
		do500(w, fmt.Errorf("QuotaAuthorization: [%w]", err))
		return
	}

	sub := resp.Response().Subscription()
	subId := sub.ID()
	userName := seat.AccountUsername

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(api.Seat{
		SubscriptionId:  &subId,
		AccountUsername: &userName,
	}); err != nil {
		do500(w, err)
		return
	}
}
