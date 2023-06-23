package app

import (
	"fmt"
	"os"

	"github.com/leighmacdonald/bd/pkg/util"
	"github.com/leighmacdonald/gbans/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func MustCreateLogger(logFile string) *zap.Logger {
	var loggingConfig zap.Config
	if config.General.Mode == config.ReleaseMode {
		loggingConfig = zap.NewProductionConfig()
		loggingConfig.DisableCaller = true
	} else {
		loggingConfig = zap.NewDevelopmentConfig()
		loggingConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	if logFile != "" {
		if util.Exists(logFile) {
			if err := os.Remove(logFile); err != nil {
				panic(fmt.Sprintf("Failed to remove log file: %v", err))
			}
		}
		loggingConfig.OutputPaths = append(loggingConfig.OutputPaths, logFile)
		// loggingConfig.Level.SetLevel(zap.DebugLevel)
	}
	l, errLogger := loggingConfig.Build()
	if errLogger != nil {
		os.Exit(1)
	}
	return l.Named("gb")
}
