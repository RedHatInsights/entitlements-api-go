package server

import (
	"fmt"
	"net/http"

	"github.com/RedHatInsights/entitlements-api-go/config"
	"github.com/RedHatInsights/entitlements-api-go/logger"
	"github.com/sirupsen/logrus"
)

// Launch the server.
func Launch() {
	r := DoRoutes()
	var port = config.GetConfig().Options.GetString(config.Keys.Port)
	logger.Log.WithFields(logrus.Fields{"port": port}).Info("server starting")
	err := http.ListenAndServe(fmt.Sprintf(":%s", port), r)
	logger.Log.WithFields(logrus.Fields{"error": err}).Fatal("server stopped")
}
