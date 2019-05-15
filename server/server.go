package server

import (
	"fmt"
	"net/http"

	"github.com/RedHatInsights/entitlements-api-go/config"
)

func Launch() {
	r := DoRoutes()
	var port string = config.GetConfig().Options.GetString("Port")
	http.ListenAndServe(fmt.Sprintf(":%s", port), r)
}
