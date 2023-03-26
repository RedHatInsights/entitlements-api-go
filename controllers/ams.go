package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/RedHatInsights/entitlements-api-go/config"
	"github.com/RedHatInsights/entitlements-api-go/openapi"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/ocm-sdk-go/logging"
)

type SeatManagerApi struct {
	sdk *sdk.Connection
}

var _ openapi.ServerInterface = &SeatManagerApi{}

func NewSeatManagerApi() *SeatManagerApi {
	logger, err := logging.NewGoLoggerBuilder().Debug(false).Build()
	if err != nil {
		panic(err)
	}

	cfg := config.GetConfig()

	clientId := cfg.Options.GetString(config.Keys.ClientID)
	secret := cfg.Options.GetString(config.Keys.ClientSecret)
	tokenUrl := cfg.Options.GetString(config.Keys.TokenURL)
	amsUrl := cfg.Options.GetString(config.Keys.AMSHost)

	client, err := sdk.NewConnectionBuilder().
		Logger(logger).
		Client(clientId, secret).
		TokenURL(tokenUrl).
		URL(amsUrl).
		BuildContext(context.Background())

	if err != nil {
		panic(err)
	}

	api := &SeatManagerApi{
		sdk: client,
	}
	return api
}

func (api *SeatManagerApi) DeleteSeatsId(w http.ResponseWriter, r *http.Request, id string) {
	// idObj := identity.Get(req.Context()).Identity
	// TODO: check incoming ident orgId against orgId of subscription
	// TODO: call DELETE subscription {id}
}

func (api *SeatManagerApi) GetSeats(w http.ResponseWriter, r *http.Request, params openapi.GetSeatsParams) {
	// TODO: call subscription search
}

func (api *SeatManagerApi) PostSeats(w http.ResponseWriter, r *http.Request) {
	seat := new(openapi.SeatRequest)
	defer r.Body.Close()
	if err := json.NewDecoder(r.Body).Decode(seat); err != nil {
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}
	fmt.Printf("%+v\n", seat)
	// TODO: call quota_cost to get quota version
	// TODO: call quota_authorization
}
