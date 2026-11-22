package httphelper

import (
	"errors"
	"log/slog"
	"net/http"
	"path/filepath"
	"slices"

	"github.com/leighmacdonald/gbans/frontend"
	"github.com/leighmacdonald/gbans/internal/log"
	"github.com/leighmacdonald/gbans/internal/opengraph"
)

var (
	ErrValidator       = errors.New("failed to register validator")
	ErrFrontendRoutes  = errors.New("failed to initialize frontend asset routes")
	ErrStaticPathError = errors.New("could not load static path")
)

type RouterOpts struct {
	HTTPLogEnabled  bool
	LogLevel        log.Level
	HTTPOtelEnabled bool
	SentryDSN       string
	Version         string
	FrontendEnable  bool
	StaticPath      string
	HTTPCORSEnabled bool
	DevMode         bool
	CORSOrigins     []string
	// OpenGraph, when set, injects opengraph tags into html responses served
	// to known crawler bots.
	OpenGraph *opengraph.Repository
}

func CreateRouter(opts RouterOpts) (*http.ServeMux, http.Handler, error) {
	var middleware []func(http.Handler) http.Handler

	middleware = append(middleware, recoveryHandler)

	if opts.OpenGraph != nil {
		middleware = append(middleware, opts.OpenGraph.Middleware)
	}

	if opts.HTTPLogEnabled {
		middleware = append(middleware, useLogging(opts.LogLevel, opts.HTTPOtelEnabled))
	}

	if opts.SentryDSN != "" {
		middleware = append(middleware, useSentry(opts.Version))
	}

	mux := http.NewServeMux()

	middleware = append(middleware, useSecure(opts.DevMode, ""))

	if opts.HTTPCORSEnabled {
		if len(opts.CORSOrigins) > 0 {
			middleware = append(middleware, useCors(opts.CORSOrigins))
		} else {
			slog.Warn("No cors origins defined, disabling")
		}
	}

	if opts.FrontendEnable {
		if opts.StaticPath == "" {
			opts.StaticPath = "./frontend/dist"
		}

		absStaticPath, errStaticPath := filepath.Abs(opts.StaticPath)
		if errStaticPath != nil {
			return nil, nil, errors.Join(errStaticPath, ErrStaticPathError)
		}

		if err := frontend.AddRoutes(mux, absStaticPath); err != nil {
			return nil, nil, errors.Join(err, ErrFrontendRoutes)
		}
	}

	var handler http.Handler = mux
	for _, mw := range slices.Backward(middleware) {
		handler = mw(handler)
	}

	return mux, handler, nil
}
