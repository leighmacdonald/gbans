package log

import (
	"errors"

	"github.com/getsentry/sentry-go"
)

var ErrClientInit = errors.New("failed to initialize sentry client")

func NewSentryClient(dsn string, tracing bool, sampleRate float64, buildVersion string, environment string) (*sentry.Client, error) {
	env := "production"
	if environment == "test" {
		env = "development"
	}

	hub := sentry.CurrentHub()
	client, errClient := sentry.NewClient(sentry.ClientOptions{
		Dsn:              dsn,
		EnableTracing:    tracing,
		TracesSampleRate: sampleRate,
		SendDefaultPII:   true,
		SampleRate:       1.0,
		Release:          buildVersion,
		Environment:      env,
	})

	if errClient != nil {
		return nil, errors.Join(errClient, ErrClientInit)
	}

	hub.BindClient(client)

	return client, nil
}
