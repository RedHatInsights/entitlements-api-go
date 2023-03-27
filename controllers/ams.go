//go:generate go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen --config=types.cfg.yaml ../apispec/api.spec.json
//go:generate go run github.com/deepmap/oapi-codegen/cmd/oapi-codegen --config=server.cfg.yaml ../apispec/api.spec.json

package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/RedHatInsights/entitlements-api-go/ams"
	"github.com/RedHatInsights/entitlements-api-go/api"
	"github.com/RedHatInsights/entitlements-api-go/logger"
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

func (s *SeatManagerApi) DeleteSeatsId(w http.ResponseWriter, r *http.Request, id string) {
	idObj := identity.Get(r.Context()).Identity
	subscription, err := s.client.GetSubscription(id)
	if err != nil {
		return // handle it?
	}
	orgId, ok := subscription.GetOrganizationID()
	if !ok {
		return // handle it!
	}

	if orgId != idObj.Internal.OrgID {
		return // Can't delete subs outside of org
	}

	if err = s.client.DeleteSubscription(id); err != nil {
		return // handle it!
	}
}

func (s *SeatManagerApi) GetSeats(w http.ResponseWriter, r *http.Request, params api.GetSeatsParams) {
	logger.Log.Info("GetSeats?")
	subs, err := s.client.GetSubscriptions()
	if err != nil {
		return
	}
	logger.Log.Info("%s", subs)
	// TODO: call subscription search
}

func (s *SeatManagerApi) PostSeats(w http.ResponseWriter, r *http.Request) {
	seat := new(api.SeatRequest)
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(seat); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	logger.Log.Infof("PostSeats: %+v", seat)
	// TODO: call quota_cost to get quota version
	// TODO: call quota_authorization
}
