package log

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"

	"github.com/dotse/slug"
)

func MustCreateLogger(debugLogPath string, levelString string) func() {
	var (
		logHandler slog.Handler
		level      slog.Level
	)

	switch levelString {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	default:
		level = slog.LevelError
	}

	closer := func() {}

	opts := slug.HandlerOptions{
		HandlerOptions: slog.HandlerOptions{
			Level: level,
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
