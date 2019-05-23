package server

import (
	"fmt"
	"net/http"

	"github.com/RedHatInsights/entitlements-api-go/config"
	"github.com/RedHatInsights/entitlements-api-go/logger"
	"go.uber.org/zap"
)

// Launch the server.
func Launch() {
	r := DoRoutes()
	var port = config.GetConfig().Options.GetString(config.Keys.Port)
	logger.Log.Info("server starting", zap.String("port", port))
	err := http.ListenAndServe(fmt.Sprintf(":%s", port), r)
	logger.Log.Fatal("server stopped",
		zap.Error(err),
	)
}
