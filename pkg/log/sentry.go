package log

import (
	"errors"

	"github.com/getsentry/sentry-go"
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
