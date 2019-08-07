package apispec

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi"
)

// OpenAPISpec responds back with the openapi spec
func OpenAPISpec(r chi.Router) {
	currDir, err := os.Getwd()
	specDir := string(currDir) + "/apispec/Aapi.spec.json"
	fmt.Println(specDir)
	specFile, err := os.Open(specDir)

	if err != nil {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("OpenApi Spec not available\n"))
			log.Panic(err)
		})
		return
	}

	byteArr, _ := ioutil.ReadAll(specFile)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write(byteArr)
	})
}
