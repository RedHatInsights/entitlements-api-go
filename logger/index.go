package logger

import (
	"flag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger
var logWithStack *zap.Logger

func Logger() *zap.Logger {
	if (log == nil) {
		logLevel := zapcore.InfoLevel
		if flag.Lookup("test.v") != nil { logLevel = zapcore.FatalLevel }

		cfg := zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			MessageKey:     "msg",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.EpochTimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}

		logger, _ := zap.Config{
			Encoding:    "json",
			Level:       zap.NewAtomicLevelAt(logLevel),
			OutputPaths: []string{"stdout"},
			ErrorOutputPaths: []string{"stderr"},
			EncoderConfig: cfg,
		}.Build()

		defer logger.Sync()
		log = logger
	}

	return log
}
