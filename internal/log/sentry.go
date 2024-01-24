package log

import (
	"errors"

	"github.com/TheZeroSlave/zapsentry"
	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var ErrClientInit = errors.New("failed to initialize sentry client")

func NewSentryClient(dsn string, tracing bool, sampleRate float64, buildVersion string) (*sentry.Client, error) {
	hub := sentry.CurrentHub()
	client, errClient := sentry.NewClient(sentry.ClientOptions{
		// "https://examplePublicKey@o0.ingest.sentry.io/0"
		Dsn:           dsn,
		EnableTracing: tracing,
		// Set TracesSampleRate to 1.0 to capture 100%		// of transactions for performance monitoring.
		// We recommend adjusting this value in production,
		TracesSampleRate: sampleRate,
		Release:          buildVersion,
	})

	if errClient != nil {
		return nil, errors.Join(errClient, ErrClientInit)
	}

	hub.BindClient(client)

	return client, nil
}

func addZapSentry(log *zap.Logger, client *sentry.Client) *zap.Logger {
	cfg := zapsentry.Configuration{
		Level:             zapcore.ErrorLevel, // when to send message to sentry
		EnableBreadcrumbs: true,               // enable sending breadcrumbs to Sentry
		BreadcrumbLevel:   zapcore.InfoLevel,  // at what level should we sent breadcrumbs to sentry, this level can't be higher than `Level`
		Tags: map[string]string{
			"component": "system",
		},
	}
	core, err := zapsentry.NewCore(cfg, zapsentry.NewSentryClientFromClient(client))
	// don't use value if error was returned. Noop core will be replaced to nil soon.
	if err != nil {
		panic(err)
	}

	log = zapsentry.AttachCoreToLogger(core, log)

	// if you have web service, create a new scope somewhere in middleware to have valid breadcrumbs.
	return log.With(zapsentry.NewScope())
}
