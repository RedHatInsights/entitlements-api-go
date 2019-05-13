package server

import (
	"net/http"
	"cloud.redhat.com/entitlements/config"
)

func Launch() {
	r := DoRoutes()
	http.ListenAndServe(config.GetConfig().Port, r)
}
