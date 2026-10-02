package server

import (
	"fmt"
	"net/http"

	"github.com/RedHatInsights/entitlements-api-go/config"
	"github.com/RedHatInsights/entitlements-api-go/logger"
	"github.com/RedHatInsights/entitlements-api-go/securitylog"
	"github.com/sirupsen/logrus"
)

// Launch the server.
func Launch() {
	r := DoRoutes()
	var port = config.GetConfig().Options.GetString(config.Keys.Port)
	// Process startup - SEC-MON-REQ-1 compliance (EOI-5 process_status)
	logger.Log.WithFields(logrus.Fields{"port": port}).WithFields(securitylog.Fields(
		"STARTUP",
		"process",
		"entitlements-api-go",
		securitylog.OutcomeSuccess,
		securitylog.ProcessPrincipal("entitlements-api-go"),
	)).Info("server starting")
	err := http.ListenAndServe(fmt.Sprintf(":%s", port), r)
	// Process shutdown failure - SEC-MON-REQ-1 compliance (EOI-5 process_status, EOI-11 warnings_or_errors)
	logger.Log.WithFields(logrus.Fields{"error": err}).WithFields(securitylog.Fields(
		"SHUTDOWN",
		"process",
		"entitlements-api-go",
		securitylog.OutcomeFailure,
		securitylog.ProcessPrincipal("entitlements-api-go"),
	)).Fatal("server stopped")
}
