package apispec

import (
	"io/ioutil"
	"net/http"
	"os"

	"github.com/go-chi/chi"
)

// OpenAPISpec responds back with the openapi spec
func OpenAPISpec(r chi.Router) {
	specFile, err := os.Open("./apispec/api.spec.json")

	if err != nil {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("OpenApi Spec not available\n"))
		})
		return
	}

	byteArr, _ := ioutil.ReadAll(specFile)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write(byteArr)
	})
}
