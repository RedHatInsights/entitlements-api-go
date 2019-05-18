package server

import (
	"fmt"
	"net/http"

	"go.uber.org/zap"
	"github.com/RedHatInsights/entitlements-api-go/config"
	"github.com/RedHatInsights/entitlements-api-go/logger"
)

// Launch the server.
func Launch() {
	r := DoRoutes()
	var port = config.GetConfig().Options.GetString("Port")
	logger.Log.Info("server starting", zap.String("port", port))
	http.ListenAndServe(fmt.Sprintf(":%s", port), r)
}
