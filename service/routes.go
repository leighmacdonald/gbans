package service

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/config"
	"net/http"
)

type Route string

const (
	routeDist            Route = "dist"
	routeHome            Route = "home"
	routeBans            Route = "bans"
	routeReport          Route = "report"
	routeProfileSettings Route = "profile_settings"
	routeAppeal          Route = "appeal"
	routeLogin           Route = "login"
	routeServers         Route = "servers"
	routeLogout          Route = "logout"
	routeLoginCallback   Route = "login_callback"

	routeAdminPeople        Route = "admin_people"
	routeAdminServers       Route = "admin_servers"
	routeAdminFilteredWords Route = "admin_filtered_words"
	routeAdminImport        Route = "admin_import"

	routeAPIServers       Route = "api_servers"
	routeAPIBans          Route = "api_bans"
	routeAPIFilteredWords Route = "api_filtered_words"
	routeAPIStats         Route = "api_stats"

	routeServerAPIPingMod Route = "sapi_ping_mod"
	routeServerAPIAuth    Route = "sapi_auth"
	routeServerAPIBan     Route = "sapi_ban"
	routeServerAPICheck   Route = "sapi_check"
	routeServerAPIMessage Route = "sapi_message"
)

func initRouter() {
	ses := sessions.Sessions("gbans", cookie.NewStore([]byte(config.HTTP.CookieKey)))
	router.Use(gin.Logger())
	// Dont use session for static assets
	router.StaticFS("/dist", http.Dir(config.HTTP.StaticPath))

	session := router.Group("")
	session.Use(ses, authMiddleWare())

	session.GET(routeRaw(string(routeHome)), onIndex())
	session.GET(routeRaw(string(routeBans)), onGetBans())
	session.GET(routeRaw(string(routeAppeal)), onGetAppeal())
	session.GET(routeRaw(string(routeProfileSettings)), onGetProfileSettings())
	session.GET(routeRaw(string(routeLoginCallback)), onOpenIDCallback())
	session.GET(routeRaw(string(routeLogin)), onGetLogin())
	session.GET(routeRaw(string(routeLogout)), onGetLogout())
	session.GET(routeRaw(string(routeServers)), onGetServers())

	// Admin
	session.GET(routeRaw(string(routeAdminFilteredWords)), onAdminFilteredWords())
	session.GET(routeRaw(string(routeAdminImport)), onGetAdminImport())
	session.GET(routeRaw(string(routeAdminServers)), onGetAdminServers())
	session.GET(routeRaw(string(routeAdminPeople)), onGetAdminPeople())

	// Client API
	session.GET(routeRaw(string(routeAPIServers)), onAPIGetServers())
	session.POST(routeRaw(string(routeAppeal)), onAPIPostAppeal())
	session.GET(routeRaw(string(routeAPIStats)), onAPIGetStats())
	session.GET(routeRaw(string(routeAPIBans)), onAPIGetBans())
	session.GET(routeRaw(string(routeAPIFilteredWords)), onAPIGetFilteredWords())
	session.POST(routeRaw(string(routeAPIBans)), onAPIPostBan())

	// Game server API
	router.POST(routeRaw(string(routeServerAPIAuth)), onSAPIPostServerAuth())
	authed := router.Group("/", checkServerAuth)
	authed.GET(string(routeServerAPIBan), onGetServerBan())
	authed.POST(string(routeServerAPICheck), onPostServerCheck())
	authed.POST(routeRaw(string(routeServerAPIMessage)), onPostLogMessage())
	authed.POST(routeRaw(string(routeServerAPIPingMod)), onPostPingMod())
}

func init() {
	routes = map[Route]string{
		routeHome:            "/",
		routeDist:            "/dist",
		routeAPIServers:      "/servers",
		routeBans:            "/bans",
		routeReport:          "/report",
		routeAppeal:          "/appeal",
		routeLogin:           "/auth/login",
		routeLoginCallback:   "/auth/callback",
		routeLogout:          "/auth/logout",
		routeProfileSettings: "/profile/settings",

		routeAdminFilteredWords: "/admin/filtered_words",
		routeAdminImport:        "/admin/import",
		routeAdminServers:       "/admin/servers",
		routeAdminPeople:        "/admin/people",

		routeAPIFilteredWords: "/api/v1/filtered_words",
		routeAPIStats:         "/api/v1/stats",
		routeAPIBans:          "/api/v1/bans",

		routeServerAPIAuth:    "/sapi/v1/auth",
		routeServerAPIBan:     "/sapi/v1/ban",
		routeServerAPICheck:   "/sapi/v1/check",
		routeServerAPIMessage: "/sapi/v1/message",
		routeServerAPIPingMod: "/sapi/v1/ping_mod",
	}
}
