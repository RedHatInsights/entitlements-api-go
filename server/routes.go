package server

import (
	"context"
	"fmt"

	chilogger "github.com/766b/chi-logger"
	"github.com/RedHatInsights/entitlements-api-go/ams"
	"github.com/RedHatInsights/entitlements-api-go/api"
	"github.com/RedHatInsights/entitlements-api-go/apispec"
	"github.com/RedHatInsights/entitlements-api-go/bop"
	"github.com/RedHatInsights/entitlements-api-go/config"
	"github.com/RedHatInsights/entitlements-api-go/controllers"
	log "github.com/RedHatInsights/entitlements-api-go/logger"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redhatinsights/platform-go-middlewares/v2/identity"
)

// DoRoutes sets up the routes used by the server.
// First, it sets up the chi router using our middleware.
// Then it does the actual routing config.

func DoRoutes() chi.Router {
	r := chi.NewRouter()

	sentryMiddleware := sentryhttp.New(sentryhttp.Options{
		Repanic: true,
	})

	r.Use(prometheusMiddleware)
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(chilogger.NewLogrusMiddleware("router", log.Log))
	r.Use(sentryMiddleware.Handle)

	// Setup identity enforcement with logging
	enforceIdentity := identity.EnforceIdentityWithLogger(func(ctx context.Context, id, msg string) {
		log.Log.WithFields(map[string]interface{}{
			"identity_header": id,
			"error":           msg,
		}).Error("Identity validation failed")
	})

	configOptions := config.GetConfig().Options

	// This is odd, but the generated code will register handlers
	// and return a http.Handler.  This is normally used with .Mount,
	// but since only part of the server is using code gen this is
	// a way to hack it in
	if !configOptions.GetBool(config.Keys.DisableSeatManager) {
		debug := configOptions.GetBool(config.Keys.Debug)

		amsClient, err := ams.NewClient(debug)
		if err != nil {
			panic(fmt.Sprintf("Error constructing ams client: [%s]", err))
		}

		bopClient, err := bop.NewClient(debug)
		if err != nil {
			panic(fmt.Sprintf("Error constructing bop client: [%s]", err))
		}

		seatManagerApi := controllers.NewSeatManagerApi(amsClient, bopClient)
		api.HandlerFromMuxWithBaseURL(seatManagerApi, r.With(enforceIdentity), "/api/entitlements/v1")
	}

	r.Route("/api/entitlements/v1", func(r chi.Router) {
		r.With(enforceIdentity).Route("/", controllers.LubDub)
		r.Route("/openapi.json", apispec.OpenAPISpec)
		r.With(enforceIdentity).Get("/services", controllers.Services())
		r.With(enforceIdentity).Get("/compliance", controllers.Compliance())
	})

	r.Route("/status", controllers.Status)
	r.Handle("/metrics", promhttp.Handler())

	return r
}
