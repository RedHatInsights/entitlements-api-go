package main

import (
	"os"
	"time"

	"github.com/RedHatInsights/entitlements-api-go/config"
	"github.com/RedHatInsights/entitlements-api-go/controllers"
	"github.com/RedHatInsights/entitlements-api-go/logger"
	"github.com/RedHatInsights/entitlements-api-go/securitylog"
	"github.com/RedHatInsights/entitlements-api-go/server"

	"github.com/sirupsen/logrus"

	"github.com/getsentry/sentry-go"
)

func main() {

	// Init the logger first thing
	logger.InitLogger()

	var dsn string = os.Getenv("GLITCHTIP_DSN")

	if dsn != "" {
		err := sentry.Init(sentry.ClientOptions{
			Dsn: dsn,
		})
		if err != nil {
			logger.Log.WithFields(logrus.Fields{"error": err}).WithFields(securitylog.Fields(
				"STARTUP",
				"glitchtip_configuration",
				"GLITCHTIP_DSN",
				securitylog.OutcomeFailure,
				securitylog.ProcessPrincipal("entitlements-api-go"),
			)).Error("Error loading Sentry SDK with GLITCHTIP_DSN")
		} else {
			logger.Log.Info("Sentry SDK initialization using Glitchtip was successful!")
		}
	} else {
		logger.Log.Info("GLITCHTIP_DSN was not set, skipping Glitchtip initialization.")
	}

	// init config here
	if err := controllers.SetBundleInfo(config.GetConfig().Options.GetString(config.Keys.BundleInfoYaml)); err != nil {
		sentry.CaptureException(err)
		// Bundle config startup failure - SEC-MON-REQ-1 compliance (EOI-5 process_status, EOI-11 warnings_or_errors)
		logger.Log.WithFields(logrus.Fields{"error": err}).WithFields(securitylog.Fields(
			"STARTUP",
			"bundle_configuration",
			config.GetConfig().Options.GetString(config.Keys.BundleInfoYaml),
			securitylog.OutcomeFailure,
			securitylog.ProcessPrincipal("entitlements-api-go"),
		)).Fatal("Error reading bundles.yml")
	}

	server.Launch()

	// Flush buffered events before the program terminates.
	// Set the timeout to the maximum duration the program can afford to wait.
	defer sentry.Flush(2 * time.Second)
}
