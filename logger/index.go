package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger
var logWithStack *zap.Logger

func Logger() *zap.Logger {
	if (log == nil) {
		logger, _ := zap.NewProduction(zap.AddStacktrace(zapcore.PanicLevel))

		defer logger.Sync()
		log = logger
	}

	return log
}

func LoggerWithStack() *zap.Logger {
	if (logWithStack == nil) {
		logger, _ := zap.NewProduction()

		defer logger.Sync()
		logWithStack = logger
	}

	return logWithStack
}
