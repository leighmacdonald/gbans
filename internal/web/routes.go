package web

import (
	"github.com/Depado/ginprom"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/leighmacdonald/gbans/internal/store"
	"github.com/leighmacdonald/gbans/internal/web/ws"
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

func (w *Web) setupRouter(r *gin.Engine, db store.Store, bot discord.ChatBot, logMsgChan chan ws.LogPayload) {
	handlers := ws.Handlers{
		ws.Sup: w.onSup,
	}
	rpcService := ws.NewService(handlers, logMsgChan)
	r.Use(gin.Logger())
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOriginFunc = func(requestedOrigin string) bool {
		for _, allowedOrigin := range config.HTTP.CorsOrigins {
			if allowedOrigin == requestedOrigin {
				return true
			}
		}
		return false
	}
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
	if config.General.Mode == config.Release {
		r.StaticFS("/dist", http.FS(content))
	} else {
		r.StaticFS("/dist", http.Dir(ap))
	}
	idxPath := filepath.Join(ap, "index.html")
	for _, rt := range []string{
		"/", "/servers", "/profile", "/bans", "/appeal", "/settings",
		"/admin/server_logs", "/admin/servers", "/admin/people", "/admin/ban", "/admin/reports",
		"/admin/import", "/admin/filters", "/404", "/logout"} {
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
	r.GET("/login/success", w.onLoginSuccess())
	r.GET("/auth/callback", w.onOpenIDCallback())
	r.GET("/api/ban/:ban_id", w.onAPIGetBanByID(db))
	r.POST("/api/bans", w.onAPIGetBans(db))
	r.GET("/api/profile", w.onAPIProfile(db))
	r.GET("/api/servers", w.onAPIGetServers(db))
	r.GET("/api/stats", w.onAPIGetStats(db))
	r.GET("/api/competitive", w.onAPIGetCompHist(db))
	r.GET("/api/filtered_words", w.onAPIGetFilteredWords(db))
	r.GET("/api/players", w.onAPIGetPlayers(db))
	r.GET("/api/auth/logout", w.onGetLogout())
	r.GET("/api/ws", rpcService.Start())

	// Game server plugin routes
	r.POST("/api/server_auth", w.onSAPIPostServerAuth(db))
	// IsServer Auth Request
	serverAuth := r.Use(w.authMiddleWare(db))
	serverAuth.POST("/api/ping_mod", w.onPostPingMod(bot))
	serverAuth.POST("/api/check", w.onPostServerCheck(db))
	serverAuth.POST("/api/demo", w.onPostDemo(db))

	// Basic logged in user
	authed := r.Use(w.authMiddleware(db, model.PAuthenticated))
	authed.GET("/api/current_profile", w.onAPICurrentProfile())
	authed.GET("/api/auth/refresh", w.onTokenRefresh())

	// Moderator access
	modRoute := r.Use(w.authMiddleware(db, model.PModerator))
	modRoute.POST("/api/ban", w.onAPIPostBanCreate())

	// Admin access
	modAdmin := r.Use(w.authMiddleware(db, model.PAdmin))
	modAdmin.POST("/api/server", w.onAPIPostServer())
}
