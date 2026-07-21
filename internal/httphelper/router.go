package httphelper

import (
	"errors"
	"log/slog"
	"net/http"
	"net/http/pprof"
	"path/filepath"
	"slices"

	"github.com/leighmacdonald/gbans/frontend"
	"github.com/leighmacdonald/gbans/internal/log"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	ErrValidator       = errors.New("failed to register validator")
	ErrFrontendRoutes  = errors.New("failed to initialize frontend asset routes")
	ErrStaticPathError = errors.New("could not load static path")
)

type RouterOpts struct {
	HTTPLogEnabled    bool
	LogLevel          log.Level
	HTTPOtelEnabled   bool
	SentryDSN         string
	Version           string
	PProfEnabled      bool
	PrometheusEnabled bool
	FrontendEnable    bool
	StaticPath        string
	HTTPCORSEnabled   bool
	CORSOrigins       []string
}

func CreateRouter(opts RouterOpts) (*http.ServeMux, http.Handler, error) {
	var middleware []func(http.Handler) http.Handler

	middleware = append(middleware, recoveryHandler)

	if opts.HTTPLogEnabled {
		middleware = append(middleware, useLogging(opts.LogLevel, opts.HTTPOtelEnabled))
	}

	if opts.SentryDSN != "" {
		middleware = append(middleware, useSentry(opts.Version))
	}

	mux := http.NewServeMux()

	if opts.PProfEnabled {
		mux.HandleFunc("GET /debug/pprof/", pprof.Index)
		mux.HandleFunc("GET /debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("GET /debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("GET /debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("GET /debug/pprof/trace", pprof.Trace)
	}

	if opts.PrometheusEnabled {
		mux.Handle("GET /metrics", promhttp.Handler())
	}

	if opts.HTTPCORSEnabled {
		if len(opts.CORSOrigins) > 0 {
			middleware = append(middleware, useSecure(false, ""))
			middleware = append(middleware, useCors(opts.CORSOrigins))
		} else {
			slog.Warn("No cors origins defined, disabling")
		}
	}

	if opts.PrometheusEnabled {
		middleware = append(middleware, usePrometheus)
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
