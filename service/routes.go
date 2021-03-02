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

	tokenAuthed := router.Use(authMiddleWare())

	tokenAuthed.GET(routeRaw(string(routeHome)), onIndex())
	tokenAuthed.GET(routeRaw(string(routeLoginCallback)), onOpenIDCallback())
	tokenAuthed.GET(routeRaw(string(routeLoginSuccess)), onLoginSuccess())

	// Client API
	tokenAuthed.GET(routeRaw(string(routeAPIServers)), onAPIGetServers())
	tokenAuthed.GET(routeRaw(string(routeAPIStats)), onAPIGetStats())
	tokenAuthed.POST(routeRaw(string(routeAPIBans)), onAPIGetBans())
	tokenAuthed.GET(routeRaw(string(routeAPIBansByID)), onAPIGetBanByID())
	tokenAuthed.GET(routeRaw(string(routeAPIFilteredWords)), onAPIGetFilteredWords())
	tokenAuthed.GET(routeRaw(string(routeAPIProfile)), onAPIProfile())
	tokenAuthed.POST(routeRaw(string(routeAPIBansCreate)), onAPIPostBanCreate())

	// Game server API
	router.POST(routeRaw(string(routeServerAPIAuth)), onSAPIPostServerAuth())
	authed := router.Group("/", checkServerAuth)
	authed.GET(string(routeServerAPIBan), onGetServerBan())
	authed.POST(string(routeServerAPICheck), onPostServerCheck())
	authed.POST(routeRaw(string(routeServerAPIMessage)), onPostLogMessage())
	authed.POST(routeRaw(string(routeServerAPIPingMod)), onPostPingMod())
	authed.POST(routeRaw(string(routeServerAPILogAdd)), onPostLogAdd())
}

func init() {
	routes = map[routeKey]string{
		routeHome:          "/",
		routeDist:          "/dist",
		routeAPIServers:    "/servers",
		routeLoginCallback: "/auth/callback",
		routeLoginSuccess:  "/login/success",

		routeAPIFilteredWords: "/api/v1/filtered_words",
		routeAPIStats:         "/api/v1/stats",
		routeAPIBans:          "/api/v1/bans",
		routeAPIBansByID:      "/api/v1/ban/:ban_id",
		routeAPIBansCreate:    "/api/v1/bans_create",
		routeAPIProfile:       "/api/v1/profile",

		routeServerAPIAuth:    "/sapi/v1/auth",
		routeServerAPIBan:     "/sapi/v1/ban",
		routeServerAPICheck:   "/sapi/v1/check",
		routeServerAPIMessage: "/sapi/v1/message",
		routeServerAPIPingMod: "/sapi/v1/ping_mod",
		routeServerAPILogAdd:  "/sapi/v1/log",
	}
}
