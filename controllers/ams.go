//go:generate go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen@v1.12.4 --config=types.cfg.yaml ../apispec/api.spec.json
//go:generate go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen@v1.12.4 --config=server.cfg.yaml ../apispec/api.spec.json

package controllers

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/RedHatInsights/entitlements-api-go/ams"
	"github.com/RedHatInsights/entitlements-api-go/api"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

type SeatManagerApi struct {
	client ams.AMSInterface
}

var _ api.ServerInterface = &SeatManagerApi{}

func NewSeatManagerApi() *SeatManagerApi {

	c, err := ams.NewClient()
	if err != nil {
		panic(err)
	}

	api := &SeatManagerApi{
		client: c,
	}
	return api
}

func NewMockSeatManagerApi() *SeatManagerApi {
	return &SeatManagerApi{
		client: &ams.TestClient{},
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

	subs, err := s.client.GetSubscriptions(limit, page)
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

	resp := api.ListSeatsResponsePagination{
		Meta: &api.PaginationMeta{
			Count: toPtr(int64(len(seats))),
		},
		Links: &api.PaginationLinks{},
		Data:  seats,
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

	quotaCost, err := s.client.GetQuotaCost(idObj.Internal.OrgID)
	if err != nil {
		do500(w, err)
		return
	}

	resp, err := s.client.QuotaAuthorization(idObj.User.Username, quotaCost.Version())
	if err != nil {
		do500(w, err)
		return
	}

	sub := resp.Response().Subscription()
	subId := sub.ID()
	userName := sub.Creator().Username()

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
