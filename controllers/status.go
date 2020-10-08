package controllers

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/RedHatInsights/entitlements-api-go/config"
	"github.com/go-chi/chi"
)

type info struct {
	Version string `json:"version"`
}

type spec struct {
	Info info `json:"info"`
}

type statusInfo struct {
	APIVersion string `json:"apiVersion"`
	Commit     string `json:"commit"`
}

func buildStatus() statusInfo {
	specFilePath := config.GetConfig().Options.GetString(config.Keys.OpenAPISpecPath)
	specFile, err := ioutil.ReadFile(specFilePath)

	apiVersion := ""

	var s spec

	if err == nil {
		err := json.Unmarshal(specFile, &s)
		if err == nil {
			apiVersion = s.Info.Version
		}
	}

	status := statusInfo{}
	status.APIVersion = apiVersion
	status.Commit = os.Getenv("OPENSHIFT_BUILD_COMMIT")

	return status
}

// Status responds back with service status information
func Status(r chi.Router) {
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(buildStatus())
	})
}
