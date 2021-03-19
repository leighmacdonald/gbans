package service

import (
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/config"
	"net/http"
)

func initRouter() {
	defaultRoute := func(c *gin.Context) {
		c.Data(200, gin.MIMEHTML, []byte(baseLayout))
	}
	router.Use(gin.Logger())
	// Dont use session for static assets
	router.StaticFS("/dist", http.Dir(config.HTTP.StaticPath))
	//router.GET(routeRaw(string(routeHome)), )
	router.NoRoute(defaultRoute)
	router.GET("/login/success", onLoginSuccess())
	router.GET("/auth/callback", onOpenIDCallback())
	router.GET("/api/ban/:ban_id", onAPIGetBanByID())
	router.POST("/api/bans", onAPIGetBans())
	router.GET("/api/profile", onAPIProfile())
	router.GET("/api/servers", onAPIGetServers())
	router.GET("/api/stats", onAPIGetStats())
	router.GET("/api/filtered_words", onAPIGetFilteredWords())

	// Server Auth Request
	router.POST("/api/server_auth", onSAPIPostServerAuth())

	tokenAuthed := router.Use(authMiddleWare())

	// Client API
	tokenAuthed.GET("/api/current_profile", onAPICurrentProfile())
	tokenAuthed.GET("/api/players", onAPIGetPlayers())
	tokenAuthed.POST("/api/ban", onAPIPostBanCreate())
	tokenAuthed.GET("/api/auth/refresh", onTokenRefresh())
	tokenAuthed.GET("/api/auth/logout", onGetLogout())

	// Game server API
	tokenAuthed.POST("/api/ping_mod", onPostPingMod())

	// Server to Server API
	router.POST("/api/log", onPostLogAdd())
	tokenAuthed.POST("/api/check", onPostServerCheck())

}
