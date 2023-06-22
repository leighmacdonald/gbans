package util

import (
	"io"

	"go.uber.org/zap"
)

func LogCloser(closer io.Closer, logger *zap.Logger) {
	if errClose := closer.Close(); errClose != nil {
		logger.Error("Failed to close", zap.Error(errClose))
	}
}
