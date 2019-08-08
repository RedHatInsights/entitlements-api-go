package apispec

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi"
)

// OpenAPISpec responds back with the openapi spec
func OpenAPISpec(r chi.Router) {
	// currDir, err := os.Getwd()
	currDir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	fmt.Println(currDir)
	specDir, _ := filepath.Abs("./apispec/api.spec.json")
	// fmt.Println(specDir)
	specFile, err := ioutil.ReadFile(specDir)

	if err != nil {
		r.Get("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("OpenApi Spec not available\n"))
			log.Panic(err)
		})
		return
	}

	//byteArr, _ := ioutil.ReadAll(specFile)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write(specFile)
	})
}
