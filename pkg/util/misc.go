package util

import (
	"io"
	"log/slog"

	"github.com/leighmacdonald/gbans/pkg/log"
	"golang.org/x/exp/constraints"
)

func LogCloser(closer io.Closer) {
	if errClose := closer.Close(); errClose != nil {
		slog.Error("Failed to close", log.ErrAttr(errClose))
	}
}

func Min[T constraints.Ordered](a, b T) T {
	if a < b {
		return a
	}

	return b
}
