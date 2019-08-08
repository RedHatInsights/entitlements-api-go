package apispec

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/RedHatInsights/entitlements-api-go/config"
	"github.com/go-chi/chi"
)

// OpenAPISpec responds back with the openapi spec
func OpenAPISpec(r chi.Router) {
	// currDir, err := os.Getwd()
	// currDir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	// fmt.Println(currDir)
	// var specDir string
	// var options = viper.New()
	// options.SetDefault(specDir, "./apispec/api.spec.json")
	// options.SetEnvPrefix("ENT")
	// options.AutomaticEnv()
	keysPath := config.GetConfig().Options.GetString(config.Keys.CaPath)
	fmt.Println(keysPath)
	pathpath := config.GetConfig().Options.GetString(config.SpecFile.FilePath)
	fmt.Println(pathpath)
	specFile, err := ioutil.ReadFile(pathpath)
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
