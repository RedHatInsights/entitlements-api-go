package logger

import (
	"os"

	"github.com/sirupsen/logrus"
)

// Log is an instance of the global logrus.Logger
var Log *logrus.Logger

// InitLogger initializes the Entitlements API logger
func InitLogger() *logrus.Logger {
	if Log == nil {
		logLevel := logrus.InfoLevel

		Log = &logrus.Logger{
			Out: os.Stdout,
			Level: logLevel,
			ReportCaller: true,
		}

		formatter := &logrus.JSONFormatter{
			FieldMap: logrus.FieldMap{
				logrus.FieldKeyTime: "ts",
				logrus.FieldKeyFunc: "caller",
				logrus.FieldKeyLevel: "level",
				logrus.FieldKeyMsg: "msg",
			},
		}

		Log.SetFormatter(formatter)
	}

	return Log
}
