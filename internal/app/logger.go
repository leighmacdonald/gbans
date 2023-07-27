package app

import (
	"fmt"
	"os"

	"github.com/leighmacdonald/bd/pkg/util"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func MustCreateLogger(conf *Config) *zap.Logger {
	var loggingConfig zap.Config
	if conf.General.Mode == ReleaseMode {
		loggingConfig = zap.NewProductionConfig()
		loggingConfig.DisableCaller = true
	} else {
		loggingConfig = zap.NewDevelopmentConfig()
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

	l, errLogger := loggingConfig.Build()
	if errLogger != nil {
		panic("Failed to create log config")
	}

	return l.Named("gb")
}
