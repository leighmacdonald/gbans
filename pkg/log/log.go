package log

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"

	"github.com/dotse/slug"
)

type Level string

const (
	Debug Level = "debug"
	Info  Level = "info"
	Warn  Level = "warn"
	Error Level = "error"
)

func ToSlogLevel(level Level) slog.Level {
	switch level {
	case Debug:
		return slog.LevelDebug
	case Info:
		return slog.LevelInfo
	case Warn:
		return slog.LevelWarn
	default:
		return slog.LevelError
	}
}

func MustCreateLogger(debugLogPath string, level Level) func() {
	var logHandler slog.Handler

	closer := func() {}

	opts := slug.HandlerOptions{
		HandlerOptions: slog.HandlerOptions{
			Level: ToSlogLevel(level),
		},
	}

	if debugLogPath != "" {
		logFile, errLogFile := os.Create(debugLogPath)
		if errLogFile != nil {
			panic(fmt.Sprintf("Failed to open logfile: %v", errLogFile))
		}

		closer = func() {
			if errClose := logFile.Close(); errClose != nil {
				panic(fmt.Sprintf("Failed to close log file: %v", errClose))
			}
		}

		logHandler = slug.NewHandler(opts, logFile)
	} else {
		logHandler = slug.NewHandler(opts, os.Stdout)
	}

	slog.SetDefault(slog.New(logHandler))

	return closer
}

func ErrAttr(err error) slog.Attr {
	return slog.Any("error", err)
}

func HandlerName(skip int) slog.Attr {
	if pc, _, _, ok := runtime.Caller(skip); ok {
		return slog.String("func", runtime.FuncForPC(pc).Name())
	}

	return slog.String("func", "unknown")
}

func Closer(closer io.Closer) {
	if errClose := closer.Close(); errClose != nil {
		slog.Error("Failed to close", ErrAttr(errClose))
	}
}
