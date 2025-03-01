package log

import (
	"errors"

	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
)

var ErrClientInit = errors.New("failed to initialize sentry client")

func NewSentryClient(dsn string, tracing bool, sampleRate float64, buildVersion string, environment string) (*sentry.Client, error) {
	// We map these to the same environment values used for frontend/node to be consistent.
	env := "production"
	if environment != gin.ReleaseMode {
		env = "development"
	}

	hub := sentry.CurrentHub()
	client, errClient := sentry.NewClient(sentry.ClientOptions{
		Dsn:           dsn,
		EnableTracing: tracing,
		// Set TracesSampleRate to 1.0 to capture 100%
		// of transactions for performance monitoring.
		// We recommend adjusting this value in production,
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
