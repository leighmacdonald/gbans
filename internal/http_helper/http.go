package http_helper

import (
	"errors"
	"net/http"
	"path/filepath"
	"runtime"

	"github.com/Depado/ginprom"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/unrolled/secure"
	"github.com/unrolled/secure/cspbuilder"
	"go.uber.org/zap"
)

func httpErrorHandler(logger *zap.Logger) gin.HandlerFunc {
	log := logger.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(c *gin.Context) {
		c.Next()

		for _, ginErr := range c.Errors {
			log.Error("Unhandled HTTP Error", zap.Error(ginErr))
		}
	}
}

func useSecure(mode domain.RunMode, cspOrigin string) gin.HandlerFunc {
	cspBuilder := cspbuilder.Builder{
		Directives: map[string][]string{
			cspbuilder.DefaultSrc: {"'self'", cspOrigin},
			cspbuilder.StyleSrc:   {"'self'", "'unsafe-inline'", "https://fonts.cdnfonts.com", "https://fonts.googleapis.com"},
			cspbuilder.ScriptSrc:  {"'self'", "'unsafe-inline'", "https://www.google-analytics.com"}, // TODO  "'strict-dynamic'", "$NONCE",
			cspbuilder.FontSrc:    {"'self'", "data:", "https://fonts.gstatic.com", "https://fonts.cdnfonts.com"},
			cspbuilder.ImgSrc:     append([]string{"'self'", "data:", "https://*.tile.openstreetmap.org", "https://*.steamstatic.com", "http://localhost:9000"}, cspOrigin),
			cspbuilder.BaseURI:    {"'self'"},
			cspbuilder.ObjectSrc:  {"'none'"},
		},
	}
	secureMiddleware := secure.New(secure.Options{
		FrameDeny:             true,
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

// jsConfig contains all the variables that we inject into the frontend at runtime.
type jsConfig struct {
	SiteName        string `json:"site_name"`
	DiscordClientID string `json:"discord_client_id"`
	DiscordLinkID   string `json:"discord_link_id"`
	// External URL used to access S3 assets. media:// links are replaces with this url
	AssetURL     string `json:"asset_url"`
	BucketDemo   string `json:"bucket_demo"`
	BucketMedia  string `json:"bucket_media"`
	BuildVersion string `json:"build_version"`
	BuildCommit  string `json:"build_commit"`
	BuildDate    string `json:"build_date"`
	SentryDSN    string `json:"sentry_dsn"`
}

func ErrorHandled(ctx *gin.Context, err error) error {
	if err != nil {
		return nil
	}

	if errors.Is(err, domain.ErrPermissionDenied) {
		HandleErrPermissionDenied(ctx)
	} else if errors.Is(err, domain.ErrNoResult) {
		HandleErrNotFound(ctx)
	} else if errors.Is(err, domain.ErrBadRequest) {
		HandleErrBadRequest(ctx)
	} else if errors.Is(err, domain.ErrDuplicate) {
		HandleErrDuplicate(ctx)
	} else if errors.Is(err, domain.ErrInvalidFormat) {
		HandleErrInvalidFormat(ctx)
	} else {
		HandleErrInternal(ctx)
	}

	return err
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

func useCors(engine *gin.Engine, log *zap.Logger, conf domain.Config) {
	engine.Use(httpErrorHandler(log), gin.Recovery())
	engine.Use(useSecure(conf.General.Mode, conf.S3.ExternalURL))

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = conf.HTTP.CorsOrigins
	corsConfig.AllowHeaders = append(corsConfig.AllowHeaders, "Authorization")
	corsConfig.AllowWildcard = false
	corsConfig.AllowCredentials = false
	engine.Use(cors.New(corsConfig))
}

func usePrometheus(engine *gin.Engine) {
	prom := ginprom.New(func(prom *ginprom.Prometheus) {
		prom.Namespace = "gbans"
		prom.Subsystem = "http"
	})
	engine.Use(prom.Instrument())
}

func useFrontend(engine *gin.Engine, conf domain.Config, version domain.BuildInfo) error {
	staticPath := conf.HTTP.StaticPath
	if staticPath == "" {
		staticPath = "./dist"
	}

	absStaticPath, errStaticPath := filepath.Abs(staticPath)
	if errStaticPath != nil {
		return errors.Join(errStaticPath, domain.ErrStaticPathError)
	}

	engine.StaticFS("/dist", http.Dir(absStaticPath))

	if conf.General.Mode != domain.TestMode {
		engine.LoadHTMLFiles(filepath.Join(absStaticPath, "index.html"))
	}

	// These should findMatch  defined in the frontend. This allows us to use the browser
	// based routing when serving the SPA.
	jsRoutes := []string{
		"/", "/servers", "/profile/:steam_id", "/bans", "/appeal", "/settings", "/report",
		"/admin/server_logs", "/admin/servers", "/admin/people", "/admin/ban/steam", "/admin/ban/cidr",
		"/admin/ban/asn", "/admin/ban/group", "/admin/reports", "/admin/news", "/admin/import", "/admin/filters",
		"/404", "/logout", "/login/success", "/report/:report_id", "/wiki", "/wiki/*slug", "/log/:match_id",
		"/logs/:steam_id", "/logs", "/ban/:ban_id", "/chatlogs", "/admin/appeals", "/login", "/pug", "/quickplay",
		"/global_stats", "/stv", "/login/discord", "/notifications", "/admin/network", "/stats",
		"/stats/weapon/:weapon_id", "/stats/player/:steam_id", "/privacy-policy", "/admin/contests",
		"/contests", "/contests/:contest_id", "/forums", "/forums/:forum_id", "/forums/thread/:forum_thread_id",
	}
	for _, rt := range jsRoutes {
		engine.GET(rt, func(ctx *gin.Context) {
			if conf.Log.SentryDSNWeb != "" {
				ctx.Header("Document-Policy", "js-profiling")
			}

			ctx.HTML(http.StatusOK, "index.html", jsConfig{
				SiteName:        conf.General.SiteName,
				DiscordClientID: conf.Discord.AppID,
				DiscordLinkID:   conf.Discord.LinkID,
				AssetURL:        conf.S3.ExternalURL,
				BucketDemo:      conf.S3.BucketDemo,
				BucketMedia:     conf.S3.BucketMedia,
				BuildVersion:    version.BuildVersion,
				BuildCommit:     version.Commit,
				BuildDate:       version.Date,
				SentryDSN:       conf.Log.SentryDSNWeb,
			})
		})
	}

	return nil
}

func CreateRouter(log *zap.Logger, conf domain.Config, version domain.BuildInfo) (*gin.Engine, error) {
	engine := gin.New()
	engine.MaxMultipartMemory = 8 << 24
	engine.Use(gin.Recovery())

	if conf.Log.SentryDSN != "" {
		useSentry(engine, version.BuildVersion)
	}

	if conf.General.Mode != domain.ReleaseMode {
		pprof.Register(engine)
	}

	if conf.General.Mode != domain.TestMode {
		useCors(engine, log, conf)
	}

	// TODO add config toggle
	usePrometheus(engine)

	if err := useFrontend(engine, conf, version); err != nil {
		return nil, err
	}

	return engine, nil
}
