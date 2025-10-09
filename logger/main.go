package logger

import (
	"flag"
	"io"
	"os"
	"time"

	"github.com/RedHatInsights/entitlements-api-go/config"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	lc "github.com/redhatinsights/platform-go-middlewares/v2/logging/cloudwatch"
	"github.com/sirupsen/logrus"
)

// Log is an instance of the global logrus.Logger
var Log *logrus.Logger

// InitLogger initializes the Entitlements API logger
func InitLogger() *logrus.Logger {
	if Log == nil {
		test := flag.Lookup("test.v") != nil
		confOpts := config.GetConfig().Options

		logLevel := confOpts.GetString(config.Keys.LogLevel)
		logrusLogLevel, err := logrus.ParseLevel(logLevel)
		if err != nil {
			panic(err)
		}

		cwKey := confOpts.GetString(config.Keys.CwKey)
		cwSecret := confOpts.GetString(config.Keys.CwSecret)
		cwRegion := confOpts.GetString(config.Keys.CwRegion)
		cwLogGroup := confOpts.GetString(config.Keys.CwLogGroup)
		cwLogStream := confOpts.GetString(config.Keys.CwLogStream)

		Log = &logrus.Logger{
			Out:          os.Stdout,
			Level:        logrusLogLevel,
			ReportCaller: true,
			Hooks:        make(logrus.LevelHooks),
		}

		// Disable app logs while running tests
		if test {
			Log.Out = io.Discard
		}

		formatter := &logrus.JSONFormatter{
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "ts",
				logrus.FieldKeyFunc:  "caller",
				logrus.FieldKeyLevel: "logLevel",
				logrus.FieldKeyMsg:   "msg",
			},
		}

		Log.SetFormatter(formatter)

		if cwKey != "" && !test {
			cred := credentials.NewStaticCredentials(cwKey, cwSecret, "")
			awsconf := aws.NewConfig().WithRegion(cwRegion).WithCredentials(cred)
			writer, err := lc.NewBatchWriterWithDuration(cwLogGroup, cwLogStream, awsconf, 10*time.Second)

			if err != nil {
				Log.Info(err)
				return nil
			}

			hook := lc.NewLogrusHook(writer)
			Log.Hooks.Add(hook)
		}
	}

	return Log
}
