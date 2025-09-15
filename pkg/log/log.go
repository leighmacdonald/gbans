package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"

	"github.com/dotse/slug"
	sentryslog "github.com/getsentry/sentry-go/slog"
	slogmulti "github.com/samber/slog-multi"
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

func MustCreateLogger(ctx context.Context, debugLogPath string, level Level, useSentry bool, version string) func() {
	closer := func() {}

	opts := slug.HandlerOptions{
		HandlerOptions: slog.HandlerOptions{
			Level: ToSlogLevel(level),
		},
	}

	var handlers []slog.Handler
	if useSentry {
		handlers = append(handlers, sentryslog.Option{
			Level:     slog.LevelDebug,
			AddSource: true,
		}.NewSentryHandler(ctx))
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

		handlers = append(handlers, slug.NewHandler(opts, logFile))
	} else {
		handlers = append(handlers, slug.NewHandler(opts, os.Stdout))
	}

	defaultLogger := slog.New(slogmulti.Fanout(handlers...))

	if version != "" {
		defaultLogger.With("release", version)
	}

	slog.SetDefault(defaultLogger)

	return closer
}

func ErrAttr(err error) slog.Attr {
	return slog.Any("reason", err)
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
