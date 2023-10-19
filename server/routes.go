package server

import (
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
	"github.com/redhatinsights/platform-go-middlewares/identity"
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

	configOptions := config.GetConfig().Options
	debug := configOptions.GetBool(config.Keys.Debug)
	amsClient, err := ams.NewClient(debug)
	amsErrMap := ams.NewErrorMapper(config.GetConfig())

	if err != nil {
		panic(fmt.Sprintf("Error constructing ams client: [%s]", err))
	}

	bopClient, err := bop.NewClient(debug)
	if err != nil {
		panic(fmt.Sprintf("Error constructing bop client: [%s]", err))
	}

	// This is odd, but the generated code will register handlers
	// and return a http.Handler.  This is normally used with .Mount,
	// but since only part of the server is using code gen this is
	// a way to hack it in
	if !configOptions.GetBool(config.Keys.DisableSeatManager) {
		seatManagerApi := controllers.NewSeatManagerApi(amsClient, bopClient, amsErrMap)
		api.HandlerFromMuxWithBaseURL(seatManagerApi, r.With(identity.EnforceIdentity), "/api/entitlements/v1")
	}

	r.Route("/api/entitlements/v1", func(r chi.Router) {
		r.With(identity.EnforceIdentity).Route("/", controllers.LubDub)
		r.Route("/openapi.json", apispec.OpenAPISpec)
		r.With(identity.EnforceIdentity).Get("/services", controllers.Index())
		r.With(identity.EnforceIdentity).Get("/compliance", controllers.Compliance())
	})

	r.Route("/status", controllers.Status)
	r.Handle("/metrics", promhttp.Handler())

	return r
}
