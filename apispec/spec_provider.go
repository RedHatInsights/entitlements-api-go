package apispec

import (
	"io/ioutil"
	"log"
	"net/http"

	"github.com/RedHatInsights/entitlements-api-go/config"
	"github.com/go-chi/chi"
)

// OpenAPISpec responds back with the openapi spec
func OpenAPISpec(r chi.Router) {
	specFilePath := config.GetConfig().Options.GetString(config.Keys.OpenAPISpecPath)
	specFile, err := ioutil.ReadFile(specFilePath)
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
