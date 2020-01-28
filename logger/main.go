package logger

import (
	"os"
	"flag"
	"io/ioutil"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	lc "github.com/redhatinsights/platform-go-middlewares/logging/cloudwatch"
	"github.com/RedHatInsights/entitlements-api-go/config"
)

// Log is an instance of the global logrus.Logger
var Log *logrus.Logger

// InitLogger initializes the Entitlements API logger
func InitLogger() *logrus.Logger {
	if Log == nil {
		logLevel := logrus.InfoLevel
		test := flag.Lookup("test.v") != nil
		confOpts := config.GetConfig().Options

		cwKey := confOpts.GetString(config.Keys.CwKey)
		cwSecret := confOpts.GetString(config.Keys.CwSecret)
		cwRegion := confOpts.GetString(config.Keys.CwRegion)
		cwLogGroup := confOpts.GetString(config.Keys.CwLogGroup)
		cwLogStream := confOpts.GetString(config.Keys.CwLogStream)

		Log = &logrus.Logger{
			Out:          os.Stdout,
			Level:        logLevel,
			ReportCaller: true,
			Hooks:        make(logrus.LevelHooks),
		}

		// Disable app logs while running tests
		if test {
			Log.Out = ioutil.Discard
		}

		formatter := &logrus.JSONFormatter{
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime:  "ts",
				logrus.FieldKeyFunc:  "caller",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg:   "msg",
			},
		}

		Log.SetFormatter(formatter)

		if cwKey != "" && !test {
			cred := credentials.NewStaticCredentials(cwKey, cwSecret, "")
			awsconf := aws.NewConfig().WithRegion(cwRegion).WithCredentials(cred)
			hook, err := lc.NewBatchingHook(cwLogGroup, cwLogStream, awsconf, 10*time.Second)

			if err != nil {
				Log.Info(err)
				return nil
			}

			Log.Hooks.Add(hook)
		}
	}

	return Log
}
