package apispec

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/go-chi/chi"
)

// OpenApiSpec responds back with the openapi spec
func OpenApiSpec(r chi.Router) {
	jsonFile, err := os.Open("./apispec/api.spec.json")

	if err != nil {
		fmt.Println(err)
	}

	byteVal, _ := ioutil.ReadAll(jsonFile)

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write(byteVal)
	})
}
