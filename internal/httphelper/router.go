package httphelper

import (
	"errors"
	"log/slog"
	"path/filepath"

	"github.com/Depado/ginprom"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"github.com/leighmacdonald/gbans/frontend"
	"github.com/leighmacdonald/gbans/internal/log"
	"github.com/leighmacdonald/steamid/v4/steamid"
	sloggin "github.com/samber/slog-gin"
	"github.com/sosodev/duration"
)

var (
	ErrValidator       = errors.New("failed to register validator")
	ErrFrontendRoutes  = errors.New("failed to initialize frontend asset routes")
	ErrStaticPathError = errors.New("could not load static path")
)

type RouterOpts struct {
	HTTPLogEnabled    bool
	LogLevel          log.Level
	Mode              string
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

// CreateRouter constructs a new router using gin.Engine with the provided RouterOpts.
func CreateRouter(opts RouterOpts) (*gin.Engine, error) {
	if opts.Mode != "" {
		gin.SetMode(opts.Mode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.MaxMultipartMemory = 8 << 24
	engine.Use(recoveryHandler())
	engine.Use(errorHandler())

	if errReg := registerCustomValidators(); errReg != nil {
		return nil, errReg
	}

	if opts.HTTPLogEnabled {
		useSloggin(engine, opts.LogLevel, opts.HTTPOtelEnabled)
	}

	if opts.SentryDSN != "" {
		useSentry(engine, opts.Version)
	}

	if opts.PProfEnabled {
		pprof.Register(engine)
	}

	if opts.HTTPCORSEnabled {
		useCors(engine, opts.CORSOrigins, false)
	}

	if opts.PrometheusEnabled {
		usePrometheus(engine)
	}

	if opts.FrontendEnable {
		if err := useFrontend(engine, opts.StaticPath); err != nil {
			return nil, err
		}
	}

	return engine, nil
}

// registerCustomValidators handles registering our custom request field type validators within the
// validation engin that gin uses.
func registerCustomValidators() error {
	if instance, ok := binding.Validator.Engine().(*validator.Validate); ok {
		if err := instance.RegisterValidation("steamid", steamIDValidator); err != nil {
			return errors.Join(err, ErrValidator)
		}
		if err := instance.RegisterValidation("asnum", asNumValidator); err != nil {
			return errors.Join(err, ErrValidator)
		}
		if err := instance.RegisterValidation("duration", durationValidator); err != nil {
			return errors.Join(err, ErrValidator)
		}
	}

	return nil
}

func durationValidator(fl validator.FieldLevel) bool {
	dur, ok := fl.Field().Interface().(duration.Duration)
	if ok {
		return dur.ToTimeDuration().Seconds() > 0
	}

	return false
}

func steamIDValidator(fl validator.FieldLevel) bool {
	sid, ok := fl.Field().Interface().(steamid.SteamID)
	if ok {
		return sid.Valid()
	}

	return false
}

func asNumValidator(fl validator.FieldLevel) bool {
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

func useCors(engine *gin.Engine, origins []string, devMode bool) {
	engine.Use(useSecure(devMode, ""))

	if len(origins) > 0 {
		corsConfig := cors.DefaultConfig()
		corsConfig.AllowOrigins = origins
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
		return errors.Join(errStaticPath, ErrStaticPathError)
	}

	if errRoute := frontend.AddRoutes(engine, absStaticPath); errRoute != nil {
		return errors.Join(errRoute, ErrFrontendRoutes)
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
