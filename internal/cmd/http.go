package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/Depado/ginprom"
	"github.com/getsentry/sentry-go"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/leighmacdonald/gbans/frontend"
	"github.com/leighmacdonald/gbans/internal/auth/session"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/internal/httphelper"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
	sloggin "github.com/samber/slog-gin"
	"github.com/sosodev/duration"
	"github.com/unrolled/secure"
	"github.com/unrolled/secure/cspbuilder"
)

func errorHandler() gin.HandlerFunc {
	// To conform to rfc9457, we need to set the content-type. Calling ctx.JSON() would use the default application/json
	// content type.
	abort := func(ctx *gin.Context, apiError httphelper.APIError) {
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
		if err := ctx.Errors.Last(); err != nil {
			ctx.Abort()

			var apiError httphelper.APIError
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
				abort(ctx, httphelper.NewAPIError(http.StatusInternalServerError, httphelper.ErrInternal))
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
				log.ErrAttr(err),
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

func useCors(engine *gin.Engine, conf config.Config) {
	engine.Use(useSecure(conf.General.Mode == config.ReleaseMode, ""))

	if len(conf.HTTPCorsOrigins) > 0 {
		corsConfig := cors.DefaultConfig()
		corsConfig.AllowOrigins = conf.HTTPCorsOrigins
		corsConfig.AllowHeaders = append(corsConfig.AllowHeaders, "Authorization")
		corsConfig.ExposeHeaders = append(corsConfig.ExposeHeaders, "GBANS-AppVersion")
		corsConfig.AllowWildcard = true
		corsConfig.AllowCredentials = true

		engine.Use(cors.New(corsConfig))
	} else {
		slog.Warn("No cors origins defined, disabling")
	}
}

func usePrometheus(engine *gin.Engine) {
	prom := ginprom.New(func(prom *ginprom.Prometheus) {
		prom.Namespace = "gbans"
		prom.Subsystem = "http"
	})
	engine.Use(prom.Instrument())
}

func useFrontend(engine *gin.Engine, staticPath string) error {
	if staticPath == "" {
		staticPath = "./frontend/dist"
	}

	absStaticPath, errStaticPath := filepath.Abs(staticPath)
	if errStaticPath != nil {
		return errors.Join(errStaticPath, domain.ErrStaticPathError)
	}

	if errRoute := frontend.AddRoutes(engine, absStaticPath); errRoute != nil {
		return errors.Join(errRoute, domain.ErrFrontendRoutes)
	}

	return nil
}

func useSloggin(engine *gin.Engine, level log.Level, otelEnabled bool) {
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

	logConfig := sloggin.Config{
		DefaultLevel: logLevel,
	}

	if otelEnabled {
		logConfig.WithSpanID = true
		logConfig.WithTraceID = true
	}

	engine.Use(sloggin.NewWithConfig(slog.Default(), logConfig))
}

func recoveryHandler() gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, err any) {
		slog.Error("Recovery error:", slog.String("err", fmt.Sprintf("%v", err)))

		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Something went wrong",
		})
	})
}

var durationValidator validator.Func = func(fl validator.FieldLevel) bool {
	dur, ok := fl.Field().Interface().(duration.Duration)
	if ok {
		return dur.ToTimeDuration().Seconds() > 0
	}

	return false
}

var steamidValidator validator.Func = func(fl validator.FieldLevel) bool {
	sid, ok := fl.Field().Interface().(steamid.SteamID)
	if ok {
		return sid.Valid()
	}

	return false
}
var asNumValidator validator.Func = func(fl validator.FieldLevel) bool {
	asNum, ok := fl.Field().Interface().(int64)
	if ok {
		ranges := []struct {
			start int64
			end   int64
		}{
			{1, 23455},
			{23457, 64495},
			{131072, 4199999999},
		}

		for _, r := range ranges {
			if asNum >= r.start && asNum <= r.end {
				return true
			}
		}

		return false
	}

	return false
}

func CreateRouter(conf config.Config, version BuildInfo) (*gin.Engine, error) {
	if conf.General.Mode == config.ReleaseMode {
		gin.SetMode(gin.ReleaseMode)
	} else {
		gin.SetMode(gin.DebugMode)
	}

	engine := gin.New()
	engine.MaxMultipartMemory = 8 << 24
	engine.Use(recoveryHandler())
	engine.Use(errorHandler())

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("steamid", steamidValidator)
		v.RegisterValidation("asnum", asNumValidator)
		v.RegisterValidation("duration", durationValidator)
	}

	if conf.Log.HTTPEnabled {
		useSloggin(engine, conf.Log.Level, conf.Log.HTTPOtelEnabled)
	}

	if SentryDSN != "" {
		useSentry(engine, version.BuildVersion)
	}

	if conf.PProfEnabled {
		pprof.Register(engine)
	}

	if conf.HTTPCORSEnabled && conf.General.Mode != config.TestMode {
		useCors(engine, conf)
	}

	if conf.PrometheusEnabled {
		usePrometheus(engine)
	}

	if conf.General.Mode != config.TestMode {
		if err := useFrontend(engine, conf.HTTPStaticPath); err != nil {
			return nil, err
		}
	}

	return engine, nil
}
