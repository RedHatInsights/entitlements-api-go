package main

import (
	"github.com/RedHatInsights/entitlements-api-go/server"
	. "github.com/RedHatInsights/entitlements-api-go/logger"
)

func main() {
	// Init the logger first thing
	InitLogger()
	server.Launch()
}
