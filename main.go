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

	var dsn string = os.Getenv("SENTRY_DSN")

	if dsn != "" {
		err := sentry.Init(sentry.ClientOptions{
			// Either set your DSN here or set the SENTRY_DSN environment variable.
			//Dsn: "https://examplePublicKey@o0.ingest.sentry.io/0",
			// Either set environment and release here or set the SENTRY_ENVIRONMENT
			// and SENTRY_RELEASE environment variables.
			//Environment: "",
			//Release:     "my-project-name@1.0.0",
			// Enable printing of SDK debug messages.
			// Useful when getting started or trying to figure something out.
			Debug: false,
		})
		if err != nil {
			logger.Log.WithFields(logrus.Fields{"error": err}).Error("Error loading Sentry")
		}
	} else {
	  logger.Log.Info("SENTRY_DSN was not set, skipping Sentry initialization.")
	}

	// Init the logger first thing
	logger.InitLogger()
	// init config here
	if err := controllers.SetBundleInfo(config.GetConfig().Options.GetString(config.Keys.BundleInfoYaml)); err != nil {
		logger.Log.WithFields(logrus.Fields{"error": err}).Fatal("Error reading bundles.yml")
	}

	server.Launch()

	// Flush buffered events before the program terminates.
	// Set the timeout to the maximum duration the program can afford to wait.
	defer sentry.Flush(2 * time.Second)
}
