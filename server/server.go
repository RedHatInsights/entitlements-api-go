package server

import (
	"fmt"
	"net/http"
	"cloud.redhat.com/entitlements/config"
)

func Launch() {
	r := DoRoutes()
	var port string = config.GetConfig().Options.GetString("Port")
	http.ListenAndServe(fmt.Sprintf(":%s", port), r)
}
