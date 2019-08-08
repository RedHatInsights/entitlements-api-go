package apispec

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/spf13/viper"
)

// OpenAPISpec responds back with the openapi spec
func OpenAPISpec(r chi.Router) {
	// currDir, err := os.Getwd()
	// currDir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	// fmt.Println(currDir)
	var specDir string
	var options = viper.New()
	options.SetDefault(specDir, "./apispec/api.spec.json")
	options.AutomaticEnv()
	specFile, err := ioutil.ReadFile(options.GetString(specDir))
	fmt.Println(options.GetString(specDir))

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
