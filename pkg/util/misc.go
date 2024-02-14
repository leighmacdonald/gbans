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

// UMin64 is math.Min for uint64
func UMin64(a, b uint64) uint64 {
	if a < b {
		return a
	}

	return b
}
