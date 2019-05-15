package server

import (
	"fmt"
	"net/http"

	"github.com/RedHatInsights/entitlements-api-go/config"
)

// Launch the server.
func Launch() {
	r := DoRoutes()
	var port = config.GetConfig().Options.GetString("Port")
	http.ListenAndServe(fmt.Sprintf(":%s", port), r)
}
