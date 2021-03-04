package service

import (
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/config"
	"net/http"
)

type routeKey string

const (
	routeDist          routeKey = "dist"
	routeHome          routeKey = "home"
	routeLoginCallback routeKey = "login_callback"
	routeLoginSuccess  routeKey = "login_success"

	routeAPIServers       routeKey = "api_servers"
	routeAPIBans          routeKey = "api_bans"
	routeAPIBansByID      routeKey = "api_bans_by_id"
	routeAPIBansCreate    routeKey = "api_bans_create"
	routeAPIFilteredWords routeKey = "api_filtered_words"
	routeAPIStats         routeKey = "api_stats"
	routeAPIProfile       routeKey = "api_profile"

	routeServerAPIPingMod routeKey = "sapi_ping_mod"
	routeServerAPIAuth    routeKey = "sapi_auth"
	routeServerAPIBan     routeKey = "sapi_ban"
	routeServerAPICheck   routeKey = "sapi_check"
	routeServerAPIMessage routeKey = "sapi_message"
	routeServerAPILogAdd  routeKey = "sapi_log_add"
)

func initRouter() {
	router.Use(gin.Logger())
	// Dont use session for static assets
	router.StaticFS("/dist", http.Dir(config.HTTP.StaticPath))
	router.GET(routeRaw(string(routeHome)), func(c *gin.Context) {
		c.Data(200, gin.MIMEHTML, []byte(baseLayout))
	})

	router.GET(routeRaw(string(routeLoginSuccess)), onLoginSuccess())
	router.GET(routeRaw(string(routeLoginCallback)), onOpenIDCallback())

	tokenAuthed := router.Use(authMiddleWare())

	// Client API
	tokenAuthed.GET(routeRaw(string(routeAPIServers)), onAPIGetServers())
	tokenAuthed.GET(routeRaw(string(routeAPIStats)), onAPIGetStats())
	tokenAuthed.POST(routeRaw(string(routeAPIBans)), onAPIGetBans())
	tokenAuthed.GET(routeRaw(string(routeAPIBansByID)), onAPIGetBanByID())
	tokenAuthed.GET(routeRaw(string(routeAPIFilteredWords)), onAPIGetFilteredWords())
	tokenAuthed.GET(routeRaw(string(routeAPIProfile)), onAPIProfile())
	tokenAuthed.POST(routeRaw(string(routeAPIBansCreate)), onAPIPostBanCreate())

	// Game server API
	tokenAuthed.POST(routeRaw(string(routeServerAPIAuth)), onSAPIPostServerAuth())
	tokenAuthed.GET(string(routeServerAPIBan), onGetServerBan())
	tokenAuthed.POST(string(routeServerAPICheck), onPostServerCheck())
	tokenAuthed.POST(routeRaw(string(routeServerAPIMessage)), onPostLogMessage())
	tokenAuthed.POST(routeRaw(string(routeServerAPIPingMod)), onPostPingMod())
	tokenAuthed.POST(routeRaw(string(routeServerAPILogAdd)), onPostLogAdd())
}

func init() {
	routes = map[routeKey]string{
		routeHome:          "/",
		routeDist:          "/dist",
		routeLoginCallback: "/auth/callback",
		routeLoginSuccess:  "/login/success",

		routeAPIFilteredWords: "/api/filtered_words",
		routeAPIStats:         "/api/stats",
		routeAPIBans:          "/api/bans",
		routeAPIBansByID:      "/api/ban/:ban_id",
		routeAPIServers:       "/api/servers",
		routeAPIBansCreate:    "/api/bans_create",
		routeAPIProfile:       "/api/profile",

		routeServerAPIAuth:    "/api/auth",
		routeServerAPIBan:     "/api/ban",
		routeServerAPICheck:   "/api/check",
		routeServerAPIMessage: "/api/message",
		routeServerAPIPingMod: "/api/ping_mod",
		routeServerAPILogAdd:  "/api/log",
	}
}
