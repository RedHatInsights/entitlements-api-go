package apispec

import (
	"log"
	"net/http"
	"os"

	"github.com/RedHatInsights/entitlements-api-go/config"
	"github.com/go-chi/chi/v5"
)

// OpenAPISpec responds back with the openapi spec
func OpenAPISpec(r chi.Router) {
	specFilePath := config.GetConfig().Options.GetString(config.Keys.OpenAPISpecPath)
	specFile, err := os.ReadFile(specFilePath)
	if err != nil {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("OpenApi Spec not available\n"))
			log.Panic(err)
		})
		return
	}

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write(specFile)
	})
}
