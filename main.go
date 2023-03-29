package main

import (
	"os"
	"time"

	"github.com/RedHatInsights/entitlements-api-go/config"
	"github.com/RedHatInsights/entitlements-api-go/controllers"
	"github.com/RedHatInsights/entitlements-api-go/logger"
	"github.com/RedHatInsights/entitlements-api-go/server"

	"github.com/sirupsen/logrus"

	"github.com/getsentry/sentry-go"
)

func main() {

	// Init the logger first thing
	logger.InitLogger()

	var dsn string = os.Getenv("GLITCHTIP_DSN")

	if dsn != "" {
		err := sentry.Init(sentry.ClientOptions{})
		if err != nil {
			logger.Log.WithFields(logrus.Fields{"error": err}).Error("Error loading Sentry SDK")
		} else {
			logger.Log.Info("Sentry SDK initialization was successful!")
		}
	} else {
		logger.Log.Info("GLITCHTIP_DSN was not set, skipping Sentry initialization.")
	}

	// init config here
	if err := controllers.SetBundleInfo(config.GetConfig().Options.GetString(config.Keys.BundleInfoYaml)); err != nil {
		sentry.CaptureException(err)
		logger.Log.WithFields(logrus.Fields{"error": err}).Fatal("Error reading bundles.yml")
	}

	server.Launch()

	// Flush buffered events before the program terminates.
	// Set the timeout to the maximum duration the program can afford to wait.
	defer sentry.Flush(2 * time.Second)
}
