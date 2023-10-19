//go:generate go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen@v1.15.0 --config=types.cfg.yaml ../apispec/api.spec.json
//go:generate go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen@v1.15.0 --config=server.cfg.yaml ../apispec/api.spec.json

package controllers

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/RedHatInsights/entitlements-api-go/logger"
	v1 "github.com/openshift-online/ocm-sdk-go/accountsmgmt/v1"
	ocmErrors "github.com/openshift-online/ocm-sdk-go/errors"
	"github.com/sirupsen/logrus"

	"github.com/RedHatInsights/entitlements-api-go/ams"
	"github.com/RedHatInsights/entitlements-api-go/api"
	"github.com/RedHatInsights/entitlements-api-go/bop"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

type SeatManagerApi struct {
	ams 		ams.AMSInterface
	bop    		bop.Bop
	amsErrMapper	ams.AMSErrorMapper
}

const BASE_LINK_URL = "/api/entitlements/v1/seats"

var _ api.ServerInterface = &SeatManagerApi{}

func NewSeatManagerApi(amsClient ams.AMSInterface, bopClient bop.Bop, amsErrMapper ams.AMSErrorMapper) *SeatManagerApi {
	return &SeatManagerApi{
		ams: amsClient,
		bop: bopClient,
		amsErrMapper: amsErrMapper,
	}
}

// mapResponse will create a response based on the provided error
// if err is of a more meaningful type than error, the response status will be set to what the error dictates
// otherwise, the provided code will be used
func mapResponse(s *SeatManagerApi, err error, httpStatusCode int) api.Error {
	var amsError *ocmErrors.Error
	if errors.As(err, &amsError) {
		reason := s.amsErrMapper.MapErrorMessage(amsError)
		return api.Error{
			Error: 			toPtr(reason),
			Code:  			toPtr(amsError.Code()),
			Identifier: 	toPtr(amsError.ID()),
			OperationId: 	toPtr(amsError.OperationID()),
			Status: 		toPtr(amsError.Status()),
		}
	} 
	
	var clientError *ams.ClientError
	if errors.As(err, &clientError) {
		return api.Error{
			Error: 	toPtr(clientError.Error()),
			Status: toPtr(clientError.StatusCode),
		}
	} 

	var userDetailErr *bop.UserDetailError
	if errors.As(err, &userDetailErr) {
		return api.Error{
			Error: 	toPtr(userDetailErr.Error()),
			Status: toPtr(userDetailErr.StatusCode),
		}
	}

	return api.Error{
		Error: toPtr(err.Error()),
		Status: toPtr(httpStatusCode),
	}
}

// doError will construct an api.Error reponse and write it to the response writer
func doError(w http.ResponseWriter, s *SeatManagerApi, httpStatusCode int, err error, source string) {
	response := mapResponse(s, err, httpStatusCode)

	log := logger.Log.WithFields(logrus.Fields{"error": err, "status": httpStatusCode, "source": source})
	if *response.Status == http.StatusInternalServerError {
		log.Error("ams internal server error")
	} else {
		log.Debug("ams request error")
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(*response.Status)
	json.NewEncoder(w).Encode(response)
}

func (s *SeatManagerApi) DeleteSeatsId(w http.ResponseWriter, r *http.Request, id string) {
	idObj := identity.Get(r.Context()).Identity

	if !idObj.User.OrgAdmin {
		doError(w, s, http.StatusForbidden, fmt.Errorf("Not allowed to delete subscription %s. User must be org admin", id), "")
		return
	}

	subscription, err := s.ams.GetSubscription(id)
	if err != nil {
		doError(w, s, http.StatusInternalServerError, err, "AMS GetSubscription")
		return
	}

	subOrgId, ok := subscription.GetOrganizationID()
	if !ok {
		doError(w, s, http.StatusInternalServerError, 
			fmt.Errorf("Subscription with id [%s] does not have a corresponding ams org id, cannot verify subscription org", id), "")
		return
	}

	amsUserOrgId, err := s.ams.ConvertUserOrgId(idObj.Internal.OrgID)
	if err != nil {
		doError(w, s, http.StatusInternalServerError, err, "AMS ConvertUserOrgId")
		return
	}

	if subOrgId != amsUserOrgId {
		doError(w, s, http.StatusForbidden,
			fmt.Errorf("Not allowed to delete subscription %s. Subscription org [%s] must match user ams org id [%s]}. User org [%s]",
				id, subOrgId, amsUserOrgId, idObj.Internal.OrgID), "")
		return
	}

	if err = s.ams.DeleteSubscription(id); err != nil {
		doError(w, s, http.StatusInternalServerError, err, "AMS DeleteSubscription")
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
		doError(w, s, http.StatusBadRequest, fmt.Errorf("limit must be > 0"), "")
		return
	}

	if offset < 0 {
		doError(w, s, http.StatusBadRequest, fmt.Errorf("offset must be >= 0"), "")
		return
	}

	page := 1 + (offset / limit)

	subs, err := s.ams.GetSubscriptions(idObj.Internal.OrgID, params, limit, page)
	if err != nil {
		doError(w, s, http.StatusInternalServerError, err, "AMS GetSubscriptions")
		return
	}

	quotaCost, err := s.ams.GetQuotaCost(idObj.Internal.OrgID)
	if err != nil {
		doError(w, s, http.StatusInternalServerError, err, "AMS GetQuotaCost")
		return
	}

	var seats = make([]api.Seat, 0)
	subs.Each(func(sub *v1.Subscription) bool {
		creator, ok := sub.GetCreator()
		if !ok {
			logger.Log.WithFields(logrus.Fields{"warning": fmt.Sprintf("Missing creator data for subscription [%s]", sub.ID())}).Warn("missing ams creator data")
			creator, _ = v1.NewAccount().FirstName("UNKNOWN").LastName("UNKNOWN").Username("UNKNOWN").Build()
		}

		seats = append(seats, api.Seat{
			AccountUsername: 	toPtr(creator.Username()),
			SubscriptionId:  	toPtr(sub.ID()),
			Status:          	toPtr(sub.Status()),
			FirstName: 		 	toPtr(creator.FirstName()),
			LastName: 		 	toPtr(creator.LastName()),
			Email:				toPtr(creator.Email()),
		})
		return true
	})

	// AMS api for subscriptions doesn't return the information needed to calculate last
	prevOffset := offset - limit
	if prevOffset < 0 {
		prevOffset = 0
	}

	links := &api.PaginationLinks{
		First:    toPtr(fmt.Sprintf("%s/?limit=%d&offset=0", BASE_LINK_URL, limit)),
		Next:     toPtr(fmt.Sprintf("%s/?limit=%d&offset=%d", BASE_LINK_URL, limit, offset+limit)),
		Previous: toPtr(fmt.Sprintf("%s/?limit=%d&offset=%d", BASE_LINK_URL, limit, prevOffset)),
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
		doError(w, s, http.StatusInternalServerError, fmt.Errorf("Unexpected error encoding response [%w]", err), "")
		return
	}

}

func (s *SeatManagerApi) PostSeats(w http.ResponseWriter, r *http.Request) {
	idObj := identity.Get(r.Context()).Identity

	if !idObj.User.OrgAdmin {
		doError(w, s, http.StatusForbidden, fmt.Errorf("Not allowed to assign seats, must be an org admin."), "")
		return
	}

	seat := new(api.SeatRequest)
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(seat); err != nil {
		doError(w, s, http.StatusBadRequest, fmt.Errorf("PostSeats [%w]", err), "")
		return
	}

	user, err := s.bop.GetUser(seat.AccountUsername)
	if err != nil {
		doError(w, s, http.StatusInternalServerError, err, "BOP GetUser")
		return
	}

	if user.OrgId != idObj.Internal.OrgID {
		doError(w, s, http.StatusForbidden, fmt.Errorf("Not allowed to assign seats to users outside of Organization %s", idObj.Internal.OrgID), "")
		return
	}

	quotaCost, err := s.ams.GetQuotaCost(idObj.Internal.OrgID)
	if err != nil {
		doError(w, s, http.StatusInternalServerError, err, "AMS GetQuotaCost")
		return
	}

	resp, err := s.ams.QuotaAuthorization(seat.AccountUsername, quotaCost.Version())
	if err != nil {
		doError(w, s, http.StatusInternalServerError, err, "AMS QuotaAuthorization")
		return
	}

	if !resp.Allowed() {
		if len(resp.ExcessResources()) > 0 {
			doError(w, s, http.StatusConflict, fmt.Errorf("Assignment request was denied due to excessive resource requests"), "")
			return
		}
		doError(w, s, http.StatusForbidden, fmt.Errorf("Assignment request was denied"), "")
		return
	}

	sub := resp.Subscription()
	subId := sub.ID()
	userName := seat.AccountUsername

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err = json.NewEncoder(w).Encode(api.Seat{
		SubscriptionId:  &subId,
		AccountUsername: &userName,
	}); err != nil {
		doError(w, s, http.StatusInternalServerError, fmt.Errorf("Unexpected error encoding response [%w]", err), "")
		return
	}
}
