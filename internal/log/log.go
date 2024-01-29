package log

import (
	"fmt"
	"os"

	"github.com/getsentry/sentry-go"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/util"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func MustCreate(conf domain.Config, sentryClient *sentry.Client) *zap.Logger {
	var loggingConfig zap.Config
	if conf.General.Mode == domain.ReleaseMode {
		loggingConfig = zap.NewProductionConfig()
		loggingConfig.DisableCaller = true
	} else {
		loggingConfig = zap.NewDevelopmentConfig()
		loggingConfig.DisableStacktrace = true
		loggingConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	if conf.Log.File != "" {
		if util.Exists(conf.Log.File) {
			if err := os.Remove(conf.Log.File); err != nil {
				panic(fmt.Sprintf("Failed to remove log file: %v", err))
			}
		}

		// loggingConfig.Level.SetLevel(zap.DebugLevel)
		loggingConfig.OutputPaths = append(loggingConfig.OutputPaths, conf.Log.File)
	}

	logger, errLogger := loggingConfig.Build()
	if errLogger != nil {
		panic("Failed to create log config")
	}

	if conf.Log.SentryDSN != "" && sentryClient != nil {
		logger = addZapSentry(logger, sentryClient)
	}

	return logger.Named("gb")
}
