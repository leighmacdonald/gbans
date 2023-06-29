package app

import (
	"fmt"
	"os"

	"github.com/leighmacdonald/bd/pkg/util"
	"github.com/leighmacdonald/gbans/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func MustCreateLogger(conf *config.Config) *zap.Logger {
	var loggingConfig zap.Config
	if conf.General.Mode == config.ReleaseMode {
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
		loggingConfig.OutputPaths = append(loggingConfig.OutputPaths, conf.Log.File)
		// loggingConfig.Level.SetLevel(zap.DebugLevel)
	}
	l, errLogger := loggingConfig.Build()
	if errLogger != nil {
		panic("Failed to create log config")
	}

	return l.Named("gb")
}
