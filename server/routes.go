package server

import (
	"github.com/RedHatInsights/entitlements-api-go/controllers"
	"github.com/RedHatInsights/platform-go-middlewares/identity"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	. "github.com/RedHatInsights/entitlements-api-go/logger"
	l "github.com/treastech/logger"
)

// DoRoutes sets up the routes used by the server.
// First, it sets up the chi router using our middleware.
// Then it does the actual routing config.
func DoRoutes() chi.Router {
	r := chi.NewRouter()


	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(l.Logger(Log))
	r.Use(identity.Identity)

	r.Route("/api/entitlements/v1", func(r chi.Router) {
		r.Route("/", controllers.LubDub)
		r.Get("/services", controllers.Index(nil))
	})

	return r
}
