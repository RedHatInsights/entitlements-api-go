package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	// "github.com/RedHatInsights/entitlements-api-go/config"
	"github.com/RedHatInsights/entitlements-api-go/config"
	l "github.com/RedHatInsights/entitlements-api-go/logger"
	"github.com/RedHatInsights/entitlements-api-go/types"
	"github.com/go-chi/chi"
	sdk "github.com/openshift-online/ocm-sdk-go"
	"github.com/openshift-online/ocm-sdk-go/logging"
	"github.com/redhatinsights/platform-go-middlewares/identity"
	"github.com/sirupsen/logrus"
)

func SeatManager(r chi.Router) {

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
	r.Get("/", ListSeats(client))
	r.Post("/seat", Assign(client))
	r.Delete("/seat/{id}", Unassign(client))
}

// Returns the statistics of seats and who is currently sitting in seats
func ListSeats(client *sdk.Connection) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		// get offset and length
		idObj := identity.Get(req.Context()).Identity
		queryParameters := req.URL.Query()

		limitString := queryParameters.Get("limit")
		limit, err := strconv.Atoi(limitString)
		if err != nil {
			limit = 50
		}

		offsetString := queryParameters.Get("limit")
		offset, err := strconv.Atoi(offsetString)
		if err != nil {
			offset = 0
		}

		fmt.Printf("limit: %+v\n", limit)
		fmt.Printf("offset: %+v\n", offset)
		fmt.Printf("%+v\n", idObj)

		// TODO: call subscription search
	}
}

func Assign(client *sdk.Connection) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		// idObj := identity.Get(req.Context()).Identity
		body, err := io.ReadAll(req.Body)
		if err != nil {
			errText := fmt.Sprintf("unexpected error while reading request body: %s", err)
			l.Log.WithFields(logrus.Fields{
				"error":     err,
				"operation": "ams.Assign",
			}).Error(errText)
			http.Error(w, errText, http.StatusInternalServerError)
			return
		}
		defer req.Body.Close()
		var seat types.Seat
		if err = json.Unmarshal(body, &seat); err != nil {
			errText := fmt.Sprintf("unexpected error while unmarshalling request body: %s", err)
			l.Log.WithFields(logrus.Fields{
				"error":     err,
				"operation": "ams.Assign",
			}).Error(errText)
			http.Error(w, errText, http.StatusBadRequest)
			return
		}

		fmt.Printf("%+v\n", seat)
		// TODO: call quota_cost to get quota version
		// TODO: call quota_authorization
	}
}

func Unassign(client *sdk.Connection) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		// idObj := identity.Get(req.Context()).Identity
		// TODO: check incoming ident orgId against orgId of subscription
		// TODO: call DELETE subscription {id}
	}
}
