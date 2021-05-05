package server

import (
	"github.com/RedHatInsights/entitlements-api-go/apispec"
	"github.com/RedHatInsights/entitlements-api-go/controllers"
	log "github.com/RedHatInsights/entitlements-api-go/logger"
	"github.com/redhatinsights/platform-go-middlewares/identity"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/766b/chi-logger"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	sentryhttp "github.com/getsentry/sentry-go/http"
)

// DoRoutes sets up the routes used by the server.
// First, it sets up the chi router using our middleware.
// Then it does the actual routing config.
func DoRoutes() chi.Router {
	r := chi.NewRouter()

	sentryMiddleware := sentryhttp.New(sentryhttp.Options{
		Repanic: true,
	})

	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(chilogger.NewLogrusMiddleware("router", log.Log))
	r.Use(sentryMiddleware.Handle)

	r.Route("/api/entitlements/v1", func(r chi.Router) {
		r.With(identity.EnforceIdentity).Route("/", controllers.LubDub)
		r.Route("/openapi.json", apispec.OpenAPISpec)
		r.With(identity.EnforceIdentity).Get("/services", controllers.Index())
	})

	r.Route("/status", controllers.Status)
	r.Handle("/metrics", promhttp.Handler())

	return r
}
