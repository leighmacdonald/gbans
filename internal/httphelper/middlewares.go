package httphelper

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"time"

	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/leighmacdonald/gbans/internal/log"
	"github.com/unrolled/secure"
	"github.com/unrolled/secure/cspbuilder"
)

func recoveryHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("Recovery error:", slog.String("err", fmt.Sprintf("%v", rec)))
				RespondProblemJSON(res, http.StatusInternalServerError, APIError{
					Title: "Something went wrong",
				})
			}
		}()

		next.ServeHTTP(res, req)
	})
}

func useSecure(devMode bool, cspOrigin string) func(http.Handler) http.Handler {
	defaultSrc := []string{"'self'"}
	if cspOrigin != "" {
		defaultSrc = append(defaultSrc, cspOrigin)
	}

	cspBuilder := cspbuilder.Builder{
		Directives: map[string][]string{
			cspbuilder.DefaultSrc: defaultSrc,
			cspbuilder.StyleSrc:   {"'self'", "'unsafe-inline'", "https://fonts.cdnfonts.com", "https://fonts.googleapis.com"},
			cspbuilder.ScriptSrc:  {"'self'", "https://www.google-analytics.com", "https://browser.sentry-cdn.com/*", "https://static.cloudflareinsights.com/*"},
			cspbuilder.FontSrc:    {"'self'", "data:", "https://fonts.gstatic.com", "https://fonts.cdnfonts.com"},
			cspbuilder.ImgSrc:     append([]string{"'self'", "data:", "https://*.tile.openstreetmap.org", "https://*.steamstatic.com", "https://*.patreonusercontent.com", "http://localhost:9000"}, cspOrigin),
			cspbuilder.BaseURI:    {"'self'"},
			cspbuilder.ObjectSrc:  {"'none'"},
		},
	}

	secureMiddleware := secure.New(secure.Options{
		FrameDeny:             false,
		ContentTypeNosniff:    true,
		ContentSecurityPolicy: cspBuilder.MustBuild(),
		IsDevelopment:         devMode,
	})

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			if err := secureMiddleware.Process(res, req); err != nil {
				return
			}

			next.ServeHTTP(res, req)
		})
	}
}

func useSentry(version string) func(http.Handler) http.Handler {
	sentryHandler := sentryhttp.New(sentryhttp.Options{Repanic: true})

	return func(next http.Handler) http.Handler {
		return sentryHandler.Handle(http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			if hub := sentry.GetHubFromContext(req.Context()); hub != nil {
				hub.Scope().SetTag("version", version)
			}

			next.ServeHTTP(res, req)
		}))
	}
}

func useCors(origins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			origin := req.Header.Get("Origin")
			if slices.Contains(origins, origin) {
				res.Header().Set("Access-Control-Allow-Origin", origin)
			}

			res.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			res.Header().Set("Access-Control-Expose-Headers", "GBANS-AppVersion")
			res.Header().Set("Access-Control-Allow-Credentials", "true")

			if req.Method == http.MethodOptions {
				res.WriteHeader(http.StatusNoContent)

				return
			}

			next.ServeHTTP(res, req)
		})
	}
}

func usePrometheus(next http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		start := time.Now()
		next.ServeHTTP(res, req)
		_ = time.Since(start).Seconds()
	})
}

func useLogging(level log.Level, _ bool) func(http.Handler) http.Handler {
	logLevel := slog.LevelError
	switch level {
	case "error":
		logLevel = slog.LevelError
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "info":
		logLevel = slog.LevelInfo
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			start := time.Now()
			next.ServeHTTP(res, req)
			slog.Log(req.Context(), logLevel, "http request",
				slog.String("method", req.Method),
				slog.String("path", req.URL.Path),
				slog.Duration("duration", time.Since(start)),
			)
		})
	}
}

func encodeJSON(res http.ResponseWriter, status int, v any) {
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(status)
	if err := json.NewEncoder(res).Encode(v); err != nil {
		slog.Error("Failed to encode JSON response", slog.String("error", err.Error()))
	}
}
