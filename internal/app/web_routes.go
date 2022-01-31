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
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

var registered = false

func (w *web) setupRouter(db store.Store, r *gin.Engine) {
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = config.HTTP.CorsOrigins
	corsConfig.AllowHeaders = []string{"*"}
	corsConfig.AllowWildcard = true
	corsConfig.AllowCredentials = true
	corsConfig.AddAllowMethods("OPTIONS")
	r.Use(cors.New(corsConfig))

	if !registered {
		prom := ginprom.New(func(p *ginprom.Prometheus) {
			p.Namespace = "gbans"
			p.Subsystem = "http"
		})
		r.Use(prom.Instrument())
		registered = true
	}
	staticPath := config.HTTP.StaticPath
	if staticPath == "" {
		staticPath = "internal/web/dist"
	}
	ap, err := filepath.Abs(staticPath)
	if err != nil {
		log.Fatalf("Invalid static path")
	}
	// Don't use session for static assets
	// Note that we only use embedded assets for !release modes
	// This is to allow us the ability to develop the frontend without needing to
	// compile+re-embed the assets on each change.
	//if config.General.Mode == config.ReleaseMode {
	//	r.StaticFS("/dist", http.FS(content))
	//} else {
	//	r.StaticFS("/dist", http.Dir(ap))
	//}
	r.StaticFS("/dist", http.Dir(ap))
	idxPath := filepath.Join(ap, "index.html")

	// These should match routes defined in the frontend. This allows us to use the browser
	// based routing when serving the SPA.
	jsRoutes := []string{
		"/", "/servers", "/profile/:steam_id", "/bans", "/appeal", "/settings", "/report",
		"/admin/server_logs", "/admin/servers", "/admin/people", "/admin/ban", "/admin/reports",
		"/admin/import", "/admin/filters", "/404", "/logout", "/login/success", "/report/:report_id"}
	for _, rt := range jsRoutes {
		r.GET(rt, func(c *gin.Context) {
			idx, err := os.ReadFile(idxPath)
			if err != nil {
				c.AbortWithStatus(http.StatusInternalServerError)
				log.Errorf("Failed to load index.html")
				return
			}
			c.Data(200, "text/html", idx)
		})
	}

	r.GET("/metrics", prometheusHandler())
	r.GET("/auth/callback", w.onOpenIDCallback(db))
	r.GET("/api/ban/:ban_id", w.onAPIGetBanByID(db))
	r.POST("/api/bans", w.onAPIGetBans(db))
	r.POST("/api/appeal", w.onAPIPostAppeal(db))
	r.POST("/api/appeal/:ban_id", w.onAPIGetAppeal(db))
	r.GET("/api/profile", w.onAPIProfile(db))
	r.GET("/api/servers", w.onAPIGetServers(db))
	r.GET("/api/stats", w.onAPIGetStats(db))
	r.GET("/api/competitive", w.onAPIGetCompHist())
	r.GET("/api/filtered_words", w.onAPIGetFilteredWords(db))
	r.GET("/api/players", w.onAPIGetPlayers(db))
	r.GET("/api/auth/logout", w.onGetLogout())

	// Service discovery endpoints
	r.GET("/api/sd/prometheus/hosts", w.onAPIGetPrometheusHosts(db))
	r.GET("/api/sd/ansible/hosts", w.onAPIGetPrometheusHosts(db))

	// Game server plugin routes
	r.POST("/api/server_auth", w.onSAPIPostServerAuth(db))

	r.GET("/api/download/report/:report_media_id", w.onAPIGetReportMedia(db))
	r.POST("/api/resolve_profile", w.onAPIGetResolveProfile(db))

	// Server Auth Request
	serverAuth := r.Use(w.authMiddleWare(db))
	serverAuth.POST("/api/ping_mod", w.onPostPingMod(db))
	serverAuth.POST("/api/check", w.onPostServerCheck(db))
	serverAuth.POST("/api/demo", w.onPostDemo(db))

	// Basic logged-in user
	authed := r.Use(authMiddleware(db, model.PAuthenticated))
	authed.GET("/api/current_profile", w.onAPICurrentProfile())
	authed.GET("/api/auth/refresh", w.onTokenRefresh())
	authed.POST("/api/report", w.onAPIPostReportCreate(db))
	authed.GET("/api/report/:report_id", w.onAPIGetReport(db))
	authed.POST("/api/reports", w.onAPIGetReports(db))
	authed.POST("/api/report/:report_id/messages", w.onAPIPostReportMessage(db))
	authed.GET("/api/report/:report_id/messages", w.onAPIGetReportMessages(db))
	authed.POST("/api/logs/query", w.onAPILogsQuery(db))

	authed.POST("/api/events", w.onAPIEvents(db))

	// Moderator access
	modRoute := r.Use(authMiddleware(db, model.PModerator))
	modRoute.POST("/api/ban", w.onAPIPostBanCreate(db))

	// Admin access
	modAdmin := r.Use(authMiddleware(db, model.PAdmin))
	modAdmin.POST("/api/server", w.onAPIPostServer())
}
