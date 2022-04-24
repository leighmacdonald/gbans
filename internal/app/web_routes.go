package app

import (
	"github.com/Depado/ginprom"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"path/filepath"
)

func prometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(ctx *gin.Context) {
		h.ServeHTTP(ctx.Writer, ctx.Request)
	}
}

var registered = false

func (web *web) setupRouter(database store.Store, engine *gin.Engine) {
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = config.HTTP.CorsOrigins
	corsConfig.AllowHeaders = []string{"*"}
	corsConfig.AllowWildcard = true
	corsConfig.AllowCredentials = true
	corsConfig.AddAllowMethods("OPTIONS")
	engine.Use(cors.New(corsConfig))
	if !registered {
		prom := ginprom.New(func(prom *ginprom.Prometheus) {
			prom.Namespace = "gbans"
			prom.Subsystem = "http"
		})
		engine.Use(prom.Instrument())
		registered = true
	}
	staticPath := config.HTTP.StaticPath
	if staticPath == "" {
		staticPath = "./dist"
	}
	absStaticPath, errStaticPath := filepath.Abs(staticPath)
	if errStaticPath != nil {
		log.Fatalf("Invalid static path: %v", errStaticPath)
	}
	// Don't use session for static assets
	// Note that we only use embedded assets for !release modes
	// This is to allow us the ability to develop the frontend without needing to
	// compile+re-embed the assets on each change.
	//if config.General.Mode == config.ReleaseMode {
	//	engine.StaticFS("/dist", http.FS(content))
	//} else {
	//	engine.StaticFS("/dist", http.Dir(absStaticPath))
	//}
	engine.StaticFS("/dist", http.Dir(absStaticPath))
	idxPath := filepath.Join(absStaticPath, "index.html")

	// These should match routes defined in the frontend. This allows us to use the browser
	// based routing when serving the SPA.
	jsRoutes := []string{
		"/", "/servers", "/profile/:steam_id", "/bans", "/appeal", "/settings", "/report",
		"/admin/server_logs", "/admin/servers", "/admin/people", "/admin/ban", "/admin/reports", "/admin/news",
		"/admin/import", "/admin/filters", "/404", "/logout", "/login/success", "/report/:report_id"}
	for _, rt := range jsRoutes {
		engine.GET(rt, func(c *gin.Context) {
			idx, errRead := os.ReadFile(idxPath)
			if errRead != nil {
				c.AbortWithStatus(http.StatusInternalServerError)
				log.Errorf("Failed to load index.html from %s", idxPath)
				return
			}
			c.Data(200, "text/html", idx)
		})
	}

	engine.GET("/metrics", prometheusHandler())
	engine.GET("/auth/callback", web.onOpenIDCallback(database))
	engine.GET("/api/ban/:ban_id", web.onAPIGetBanByID(database))
	engine.POST("/api/bans", web.onAPIGetBans(database))
	engine.POST("/api/appeal", web.onAPIPostAppeal(database))
	engine.POST("/api/appeal/:ban_id", web.onAPIGetAppeal(database))
	engine.GET("/api/profile", web.onAPIProfile(database))
	engine.GET("/api/servers", web.onAPIGetServers(database))
	engine.GET("/api/stats", web.onAPIGetStats(database))
	engine.GET("/api/competitive", web.onAPIGetCompHist())
	engine.GET("/api/filtered_words", web.onAPIGetFilteredWords(database))
	engine.GET("/api/players", web.onAPIGetPlayers(database))
	engine.GET("/api/auth/logout", web.onGetLogout())
	engine.POST("/api/news_latest", web.onAPIGetNewsLatest(database))

	// Service discovery endpoints
	engine.GET("/api/sd/prometheus/hosts", web.onAPIGetPrometheusHosts(database))
	engine.GET("/api/sd/ansible/hosts", web.onAPIGetPrometheusHosts(database))

	// Game server plugin routes
	engine.POST("/api/server_auth", web.onSAPIPostServerAuth(database))

	engine.GET("/api/download/report/:report_media_id", web.onAPIGetReportMedia(database))
	engine.POST("/api/resolve_profile", web.onAPIGetResolveProfile(database))

	// Server Auth Request
	serverAuth := engine.Use(web.authMiddleWare(database))
	serverAuth.POST("/api/ping_mod", web.onPostPingMod(database))
	serverAuth.POST("/api/check", web.onPostServerCheck(database))
	serverAuth.POST("/api/demo", web.onPostDemo(database))

	// Basic logged-in user
	authed := engine.Use(authMiddleware(database, model.PAuthenticated))
	authed.GET("/api/current_profile", web.onAPICurrentProfile())
	authed.GET("/api/auth/refresh", web.onTokenRefresh())
	authed.POST("/api/report", web.onAPIPostReportCreate(database))
	authed.GET("/api/report/:report_id", web.onAPIGetReport(database))
	authed.POST("/api/reports", web.onAPIGetReports(database))
	authed.POST("/api/report/:report_id/messages", web.onAPIPostReportMessage(database))
	authed.GET("/api/report/:report_id/messages", web.onAPIGetReportMessages(database))
	authed.POST("/api/logs/query", web.onAPILogsQuery(database))

	authed.POST("/api/events", web.onAPIEvents(database))

	// Moderator access
	modRoute := engine.Use(authMiddleware(database, model.PModerator))
	modRoute.POST("/api/ban", web.onAPIPostBanCreate(database))
	modRoute.POST("/api/report/:report_id/state", web.onAPIPostBanState(database))

	// Admin access
	adminRoute := engine.Use(authMiddleware(database, model.PAdmin))
	adminRoute.POST("/api/server", web.onAPIPostServer(database))
	adminRoute.POST("/api/news", web.onAPIPostNewsCreate(database))
	adminRoute.POST("/api/news/:news_id", web.onAPIPostNewsUpdate(database))
	adminRoute.POST("/api/news_all", web.onAPIGetNewsAll(database))
}
