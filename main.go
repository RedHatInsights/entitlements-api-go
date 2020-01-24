package main

import (
	"github.com/RedHatInsights/entitlements-api-go/config"
	"github.com/RedHatInsights/entitlements-api-go/controllers"
	"github.com/RedHatInsights/entitlements-api-go/logger"
	"github.com/RedHatInsights/entitlements-api-go/server"

	"github.com/sirupsen/logrus"
)

func main() {
	// Init the logger first thing
	logger.InitLogger()
	// init config here
	if err := controllers.SetBundleInfo(config.GetConfig().Options.GetString(config.Keys.BundleInfoYaml)); err != nil {
		logger.Log.WithFields(logrus.Fields{"error": err}).Fatal("Error reading bundles.yml")
	}

	server.Launch()
}
