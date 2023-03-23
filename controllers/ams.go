package controllers

import (
	"fmt"
	"net/http"
	"strconv"

	// "github.com/RedHatInsights/entitlements-api-go/config"
	"github.com/RedHatInsights/entitlements-api-go/controllers/internal/ocm"
	"github.com/go-chi/chi"
	"github.com/redhatinsights/platform-go-middlewares/identity"
)

func SeatManager(r chi.Router) {
	client, err := ocm.NewOcmClient()
	if err != nil {
		panic(err)
	}
	r.Get("/", ListSeats(client))
	r.Post("/seat", Assign(client))
	r.Delete("/seat/{id}", Unassign(client))
}

// Returns the statistics of seats and who is currently sitting in seats
func ListSeats(client ocm.OCM) func(http.ResponseWriter, *http.Request) {
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

func Assign(client ocm.OCM) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		// idObj := identity.Get(req.Context()).Identity
		// TODO: call quota_cost to get quota version
		// TODO: call quota_authorization
	}
}

func Unassign(client ocm.OCM) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		// idObj := identity.Get(req.Context()).Identity
		// TODO: check incoming ident orgId against orgId of subscription
		// TODO: call DELETE subscription {id}
	}
}
