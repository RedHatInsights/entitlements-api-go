package main

import (
	"log"

	"github.com/RedHatInsights/entitlements-api-go/config"
	"github.com/RedHatInsights/entitlements-api-go/controllers"
	"github.com/RedHatInsights/entitlements-api-go/logger"
	"github.com/RedHatInsights/entitlements-api-go/server"
)

func main() {
	// Init the logger first thing
	logger.InitLogger()
	// init config here
	if err := controllers.SetBundleInfo(config.GetConfig().Options.GetString(config.Keys.BundleInfoYaml)); err != nil {
		log.Fatal("blow up the outside world")
	}
	server.Launch()
}
