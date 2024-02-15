package util

import (
	"io"

	"go.uber.org/zap"
	"golang.org/x/exp/constraints"
)

func LogCloser(closer io.Closer, logger *zap.Logger) {
	if errClose := closer.Close(); errClose != nil {
		logger.Error("Failed to close", zap.Error(errClose))
	}
}

func Min[T constraints.Ordered](a, b T) T {
	if a < b {
		return a
	}

	return b
}
