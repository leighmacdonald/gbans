package httphelper

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/leighmacdonald/gbans/internal/log"
	"github.com/unrolled/secure"
	"github.com/unrolled/secure/cspbuilder"
)

func recoveryHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("Recovery error:", slog.String("err", fmt.Sprintf("%v", rec)))
				RespondProblemJSON(w, http.StatusInternalServerError, APIError{
					Title: "Something went wrong",
				})
			}
		}()

		next.ServeHTTP(w, r)
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
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := secureMiddleware.Process(w, r); err != nil {
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func useSentry(version string) func(http.Handler) http.Handler {
	sentryHandler := sentryhttp.New(sentryhttp.Options{Repanic: true})

	return func(next http.Handler) http.Handler {
		return sentryHandler.Handle(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if hub := sentry.GetHubFromContext(r.Context()); hub != nil {
				hub.Scope().SetTag("version", version)
			}

			next.ServeHTTP(w, r)
		}))
	}
}

func useCors(origins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			for _, allowed := range origins {
				if origin == allowed {
					w.Header().Set("Access-Control-Allow-Origin", origin)
					break
				}
			}

			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Expose-Headers", "GBANS-AppVersion")
			w.Header().Set("Access-Control-Allow-Credentials", "true")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func usePrometheus(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
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
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			next.ServeHTTP(w, r)
			slog.Log(r.Context(), logLevel, "http request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Duration("duration", time.Since(start)),
			)
		})
	}
}

func encodeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
