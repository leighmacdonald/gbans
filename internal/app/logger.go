package app

import (
	"fmt"
	"github.com/leighmacdonald/bd/pkg/util"
	"go.uber.org/zap"
	"os"
)

func MustCreateLogger(logFile string) *zap.Logger {
	loggingConfig := zap.NewProductionConfig()
	if logFile != "" {
		if util.Exists(logFile) {
			if err := os.Remove(logFile); err != nil {
				panic(fmt.Sprintf("Failed to remove log file: %v", err))
			}
		}
		loggingConfig.OutputPaths = append(loggingConfig.OutputPaths, logFile)
		//loggingConfig.Level.SetLevel(zap.DebugLevel)
	}
	logger, errLogger := loggingConfig.Build()
	if errLogger != nil {
		fmt.Printf("Failed to create logger: %v\n", errLogger)
		os.Exit(1)
	}
	return logger.Named("gb")
}
