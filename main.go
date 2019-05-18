package main

import (
	"github.com/RedHatInsights/entitlements-api-go/logger"
	"github.com/RedHatInsights/entitlements-api-go/server"
)

func main() {
	// Init the logger first thing
	logger.InitLogger()
	server.Launch()
}
