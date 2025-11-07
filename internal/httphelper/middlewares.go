package httphelper

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/auth/session"
	"github.com/unrolled/secure"
	"github.com/unrolled/secure/cspbuilder"
)

func recoveryHandler() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, err any) {
		slog.Error("Recovery error:", slog.String("err", fmt.Sprintf("%v", err)))

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Something went wrong",
		})
	})
}

func errorHandler() gin.HandlerFunc {
	// To conform to rfc9457, we need to set the content-type. Calling ctx.JSON() would use the default application/json
	// content type.
	abort := func(ctx *gin.Context, apiError APIError) {
		ctx.Header("Content-Type", "application/problem+json")
		ctx.Status(apiError.Status)

		err := json.NewEncoder(ctx.Writer).Encode(apiError)
		if err != nil {
			ctx.Abort()

			return
		}
	}

	return func(ctx *gin.Context) {
		ctx.Next()

		// slog.HandlerName(2)
		if err := ctx.Errors.Last(); err != nil { //nolint:nestif
			ctx.Abort()

			var apiError APIError
			if errors.As(err, &apiError) {
				abort(ctx, apiError)
				if hub := sentrygin.GetHubFromContext(ctx); hub != nil {
					hub.WithScope(func(scope *sentry.Scope) {
						scope.SetExtra("title", apiError.Title)
						scope.SetExtra("detail", apiError.Detail)
						hub.CaptureException(apiError)
					})
				}
			} else {
				abort(ctx, NewAPIError(http.StatusInternalServerError, ErrInternal))
				if hub := sentrygin.GetHubFromContext(ctx); hub != nil {
					hub.WithScope(func(scope *sentry.Scope) {
						scope.SetLevel(sentry.LevelWarning)
						hub.CaptureException(err)
					})
				}
			}
			args := []any{
				slog.String("method", ctx.Request.Method),
				slog.String("path", ctx.Request.URL.RawPath),
				slog.String("error", err.Error()),
			}

			user, _ := session.CurrentUserProfile(ctx)
			sid := user.GetSteamID()
			if sid.Valid() {
				args = append(args, slog.String("steam_id", sid.String()))
			}

			slog.Error("Error in http handler", args...)
		}
	}
}

func useSecure(devMode bool, cspOrigin string) gin.HandlerFunc {
	defaultSrc := []string{"'self'"}
	if cspOrigin != "" {
		defaultSrc = append(defaultSrc, cspOrigin)
	}

	cspBuilder := cspbuilder.Builder{
		Directives: map[string][]string{
			cspbuilder.DefaultSrc: defaultSrc,
			cspbuilder.StyleSrc:   {"'self'", "'unsafe-inline'", "https://fonts.cdnfonts.com", "https://fonts.googleapis.com"},
			cspbuilder.ScriptSrc:  {"'self'", "'unsafe-inline'", "https://www.google-analytics.com", "https://browser.sentry-cdn.com/*"}, // TODO  "'strict-dynamic'", "$NONCE",
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

	secureFunc := func(ctx *gin.Context) {
		err := secureMiddleware.Process(ctx.Writer, ctx.Request)
		if err != nil {
			ctx.Abort()

			return
		}

		// Avoid header rewrite if response is a redirection.
		if status := ctx.Writer.Status(); status > 300 && status < 399 {
			ctx.Abort()
		}
	}

	return secureFunc
}

func useSentry(engine *gin.Engine, version string) {
	engine.Use(sentrygin.New(sentrygin.Options{Repanic: true}))
	engine.Use(func(ctx *gin.Context) {
		if hub := sentrygin.GetHubFromContext(ctx); hub != nil {
			hub.Scope().SetTag("version", version)
		}

		ctx.Next()
	})
}
