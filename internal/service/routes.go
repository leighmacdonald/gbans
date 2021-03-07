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

	routeAPIServers        routeKey = "api_servers"
	routeAPIBans           routeKey = "api_bans"
	routeAPIBansByID       routeKey = "api_bans_by_id"
	routeAPIBansCreate     routeKey = "api_bans_create"
	routeAPIFilteredWords  routeKey = "api_filtered_words"
	routeAPIStats          routeKey = "api_stats"
	routeAPIProfile        routeKey = "api_profile"
	routeAPICurrentProfile routeKey = "api_current_profile"
	routeAPIAuthRefresh    routeKey = "api_auth_refresh"
	routeAPIAuthLogout     routeKey = "api_auth_logout"
	routeServerAPIPingMod  routeKey = "sapi_ping_mod"
	routeServerAPIAuth     routeKey = "sapi_auth"
	routeServerAPIBan      routeKey = "sapi_ban"
	routeServerAPICheck    routeKey = "sapi_check"
	routeServerAPIMessage  routeKey = "sapi_message"
	routeServerAPILogAdd   routeKey = "sapi_log_add"
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
	router.GET(routeRaw(string(routeLoginSuccess)), onLoginSuccess())
	router.GET(routeRaw(string(routeLoginCallback)), onOpenIDCallback())
	router.GET(routeRaw(string(routeAPIBansByID)), onAPIGetBanByID())
	router.POST(routeRaw(string(routeAPIBans)), onAPIGetBans())
	router.GET(routeRaw(string(routeAPIProfile)), onAPIProfile())
	router.GET(routeRaw(string(routeAPIServers)), onAPIGetServers())
	router.GET(routeRaw(string(routeAPIStats)), onAPIGetStats())
	router.GET(routeRaw(string(routeAPIFilteredWords)), onAPIGetFilteredWords())
	router.GET(string(routeServerAPIBan), onGetServerBan())

	tokenAuthed := router.Use(authMiddleWare())

	// Client API
	tokenAuthed.GET(routeRaw(string(routeAPICurrentProfile)), onAPICurrentProfile())
	tokenAuthed.POST(routeRaw(string(routeAPIBansCreate)), onAPIPostBanCreate())
	tokenAuthed.GET(routeRaw(string(routeAPIAuthRefresh)), onTokenRefresh())
	tokenAuthed.GET(routeRaw(string(routeAPIAuthLogout)), onGetLogout())

	// Game server API
	tokenAuthed.POST(string(routeServerAPICheck), onPostServerCheck())
	tokenAuthed.POST(routeRaw(string(routeServerAPIAuth)), onSAPIPostServerAuth())
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

		routeAPIFilteredWords:  "/api/filtered_words",
		routeAPIStats:          "/api/stats",
		routeAPIBans:           "/api/bans",
		routeAPIBansByID:       "/api/ban/:ban_id",
		routeAPIServers:        "/api/servers",
		routeAPIBansCreate:     "/api/bans_create",
		routeAPIProfile:        "/api/profile",
		routeAPICurrentProfile: "/api/current_profile",
		routeAPIAuthRefresh:    "/api/auth/refresh",
		routeAPIAuthLogout:     "/api/auth/logout",
		routeServerAPIAuth:     "/api/auth",
		routeServerAPIBan:      "/api/ban",
		routeServerAPICheck:    "/api/check",
		routeServerAPIMessage:  "/api/message",
		routeServerAPIPingMod:  "/api/ping_mod",
		routeServerAPILogAdd:   "/api/log",
	}
}
