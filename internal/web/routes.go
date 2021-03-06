package web

import (
	"github.com/Depado/ginprom"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"net/http"
)

func prometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		h.ServeHTTP(c.Writer, c.Request)
	}
}

var registered = false

func SetupRouter(r *gin.Engine, logMsgChan chan LogPayload) {
	ws := newWebSocketState(logMsgChan)
	jsRoutes := func(c *gin.Context) {
		c.Data(200, gin.MIMEHTML, []byte(baseLayout))
	}
	r.Use(gin.Logger())
	if !registered {
		prom := ginprom.New(func(p *ginprom.Prometheus) {
			p.Namespace = "gbans"
			p.Subsystem = "http"
		})
		r.Use(prom.Instrument())
		registered = true
	}
	// Dont use session for static assets
	// Note that we only use embedded assets for !release modes
	// This is to allow us the ability to develop the frontend without needing to
	// compile+re-embed the assets on each change.
	if config.General.Mode == config.Release {
		r.StaticFS("/assets", http.FS(content))
	} else {
		r.StaticFS("/assets/dist", http.Dir(config.HTTP.StaticPath))
	}
	for _, rt := range []string{
		"/", "/servers", "/profile", "/bans", "/appeal", "/settings",
		"/admin/server_logs", "/admin/servers", "/admin/people", "/admin/ban", "/admin/reports",
		"/admin/import", "/admin/filters", "/404", "/logout"} {
		r.GET(rt, jsRoutes)
	}

	r.GET("/metrics", prometheusHandler())

	r.GET("/login/success", onLoginSuccess())
	r.GET("/auth/callback", onOpenIDCallback())
	r.GET("/api/ban/:ban_id", onAPIGetBanByID())
	r.POST("/api/bans", onAPIGetBans())
	r.GET("/api/profile", onAPIProfile())
	r.GET("/api/servers", onAPIGetServers())
	r.GET("/api/stats", onAPIGetStats())
	r.GET("/api/filtered_words", onAPIGetFilteredWords())
	r.GET("/api/players", onAPIGetPlayers())
	r.GET("/api/auth/logout", onGetLogout())
	r.GET("/ws", ws.onWSStart)

	// Game server plugin routes
	r.POST("/api/server_auth", onSAPIPostServerAuth())
	// IsServer Auth Request
	serverAuth := r.Use(authMiddleWare())
	serverAuth.POST("/api/ping_mod", onPostPingMod())
	serverAuth.POST("/api/check", onPostServerCheck())

	// Basic logged in user
	authed := r.Use(authMiddleware(model.PAuthenticated))
	authed.GET("/api/current_profile", onAPICurrentProfile())
	authed.GET("/api/auth/refresh", onTokenRefresh())

	// Moderator access
	modRoute := r.Use(authMiddleware(model.PModerator))
	modRoute.POST("/api/ban", onAPIPostBanCreate())

	// Admin access
	modAdmin := r.Use(authMiddleware(model.PAdmin))
	modAdmin.POST("/api/server", onAPIPostServer())
}
