package httphelper

import (
	"errors"
	"log/slog"
	"net/http"
	"path/filepath"

	"github.com/Depado/ginprom"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/frontend"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/log"
	sloggin "github.com/samber/slog-gin"
	"github.com/unrolled/secure"
	"github.com/unrolled/secure/cspbuilder"
)

func httpErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		for _, ginErr := range c.Errors {
			slog.Error("Unhandled HTTP Error", log.ErrAttr(ginErr))
		}
	}
}

func useSecure(mode domain.RunMode, cspOrigin string) gin.HandlerFunc {
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
		IsDevelopment:         mode != domain.ReleaseMode,
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

func ErrorHandledWithReturn(ctx *gin.Context, err error) error {
	ErrorHandled(ctx, err)

	return err
}

func ErrorHandled(ctx *gin.Context, err error) {
	if err == nil {
		return
	}

	switch {
	case errors.Is(err, domain.ErrPermissionDenied):
		HandleErrPermissionDenied(ctx)
	case errors.Is(err, domain.ErrNoResult):
		HandleErrNotFound(ctx)
	case errors.Is(err, domain.ErrBadRequest):
		HandleErrBadRequest(ctx)
	case errors.Is(err, domain.ErrDuplicate):
		HandleErrDuplicate(ctx)
	case errors.Is(err, domain.ErrInvalidFormat):
		HandleErrInvalidFormat(ctx)
	default:
		HandleErrInternal(ctx)
	}

	slog.Error("Error performing request",
		log.ErrAttr(err),
		slog.String("path", ctx.Request.RequestURI),
		slog.String("method", ctx.Request.Method),
		slog.String("agent", ctx.Request.UserAgent()))
}

func HandleErrPermissionDenied(ctx *gin.Context) {
	ResponseErr(ctx, http.StatusForbidden, domain.ErrPermissionDenied)
}

func HandleErrNotFound(ctx *gin.Context) {
	ResponseErr(ctx, http.StatusNotFound, domain.ErrNotFound)
}

func HandleErrBadRequest(ctx *gin.Context) {
	ResponseErr(ctx, http.StatusBadRequest, domain.ErrBadRequest)
}

func HandleErrInternal(ctx *gin.Context) {
	ResponseErr(ctx, http.StatusInternalServerError, domain.ErrInternal)
}

func HandleErrDuplicate(ctx *gin.Context) {
	ResponseErr(ctx, http.StatusConflict, domain.ErrDuplicate)
}

func HandleErrInvalidFormat(ctx *gin.Context) {
	ResponseErr(ctx, http.StatusUnsupportedMediaType, domain.ErrInvalidFormat)
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

func useCors(engine *gin.Engine, conf domain.Config) {
	engine.Use(httpErrorHandler(), gin.Recovery())
	engine.Use(useSecure(conf.General.Mode, ""))

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

func useFrontend(engine *gin.Engine, conf domain.Config) error {
	staticPath := conf.HTTPStaticPath
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

func useSloggin(engine *gin.Engine, config domain.Config) {
	logLevel := slog.LevelError
	switch config.Log.Level {
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

	if config.Log.HTTPOtelEnabled {
		logConfig.WithSpanID = true
		logConfig.WithTraceID = true
	}

	engine.Use(sloggin.NewWithConfig(slog.Default(), logConfig))
}

func CreateRouter(conf domain.Config, version domain.BuildInfo) (*gin.Engine, error) {
	engine := gin.New()
	engine.MaxMultipartMemory = 8 << 24
	engine.Use(gin.Recovery())

	if conf.Log.HTTPEnabled {
		useSloggin(engine, conf)
	}

	if conf.Sentry.SentryDSN != "" {
		useSentry(engine, version.BuildVersion)
	}

	if conf.PProfEnabled {
		pprof.Register(engine)
	}

	if conf.HTTPCORSEnabled && conf.General.Mode != domain.TestMode {
		useCors(engine, conf)
	}

	if conf.PrometheusEnabled {
		usePrometheus(engine)
	}

	if err := useFrontend(engine, conf); err != nil {
		return nil, err
	}

	return engine, nil
}
