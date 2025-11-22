package log

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"

	"github.com/dotse/slug"
	sentryslog "github.com/getsentry/sentry-go/slog"
	slogmulti "github.com/samber/slog-multi"
)

type Config struct {
	Level Level `json:"level"`
	// If set to a non-empty path, logs will also be written to the log file.
	File string `json:"file"`
	// Enable using the sloggin library for logging HTTP requests
	HTTPEnabled bool `json:"http_enabled"`
	// Enable support for OpenTelemetry by adding span/trace IDs
	HTTPOtelEnabled bool `json:"http_otel_enabled"`
	// Log level to use for http requests
	HTTPLevel Level `json:"http_level"`
}

type Level string

const (
	Debug Level = "debug"
	Info  Level = "info"
	Warn  Level = "warn"
	Error Level = "error"
)

// ToSlogLevel maps our levels to the equivalent slog level.
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

// MustCreateLogger creates and configures the default global log handler. Depending on configuration
// a local log file and an external sentry handler may also be created.
//
// Returns a cleanup function which should be called on program shutdown.
//
// Panics on failure to open log file for writing.
func MustCreateLogger(ctx context.Context, debugLogPath string, level Level, useSentry bool, version string) func() {
	var (
		closer = func() {}
		opts   = slug.HandlerOptions{
			HandlerOptions: slog.HandlerOptions{
				Level: ToSlogLevel(level),
			},
		}
		handlers []slog.Handler
	)
	if useSentry {
		handlers = append(handlers, sentryslog.Option{
			// EventLevel:     slog.LevelDebug,
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

func Closer(closer io.Closer) {
	if errClose := closer.Close(); errClose != nil {
		slog.Error("Failed to close", slog.String("error", errClose.Error()))
	}
}
