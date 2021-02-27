package service

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/config"
	"net/http"
)

type routeKey string

const (
	routeDist            routeKey = "dist"
	routeHome            routeKey = "home"
	routeBans            routeKey = "bans"
	routeBanPlayer       routeKey = "ban_player"
	routeReport          routeKey = "report"
	routeProfileSettings routeKey = "profile_settings"
	routeAppeal          routeKey = "appeal"
	routeLogin           routeKey = "login"
	routeServers         routeKey = "servers"
	routeLogout          routeKey = "logout"
	routeLoginCallback   routeKey = "login_callback"

	routeAdminPeople        routeKey = "admin_people"
	routeAdminServers       routeKey = "admin_servers"
	routeAdminFilteredWords routeKey = "admin_filtered_words"
	routeAdminImport        routeKey = "admin_import"

	routeAPIServers       routeKey = "api_servers"
	routeAPIBan           routeKey = "api_ban"
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
	ses := sessions.Sessions("gbans", cookie.NewStore([]byte(config.HTTP.CookieKey)))
	router.Use(gin.Logger())
	// Dont use session for static assets
	router.StaticFS("/dist", http.Dir(config.HTTP.StaticPath))

	session := router.Group("")
	session.Use(ses, authMiddleWare())

	session.GET(routeRaw(string(routeHome)), onIndex())
	session.GET(routeRaw(string(routeBans)), onGetBans())
	session.GET(routeRaw(string(routeBanPlayer)), onGetBanPlayer())
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
	session.GET(routeRaw(string(routeAPIBan)), onAPIGetBans())
	session.GET(routeRaw(string(routeAPIFilteredWords)), onAPIGetFilteredWords())
	session.GET(routeRaw(string(routeAPIProfile)), onAPIProfile())
	session.POST(routeRaw(string(routeAPIBan)), onAPIPostBan())

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
		routeHome:            "/",
		routeDist:            "/dist",
		routeAPIServers:      "/servers",
		routeBans:            "/bans",
		routeBanPlayer:       "/ban",
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
		routeAPIBan:           "/api/v1/ban",
		routeAPIProfile:       "/api/v1/profile",

		routeServerAPIAuth:    "/sapi/v1/auth",
		routeServerAPIBan:     "/sapi/v1/ban",
		routeServerAPICheck:   "/sapi/v1/check",
		routeServerAPIMessage: "/sapi/v1/message",
		routeServerAPIPingMod: "/sapi/v1/ping_mod",
		routeServerAPILogAdd:  "/sapi/v1/log",
	}
}
