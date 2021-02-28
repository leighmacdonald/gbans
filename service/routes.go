package service

import (
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/config"
	"net/http"
)

type routeKey string

const (
	routeDist               routeKey = "dist"
	routeHome               routeKey = "home"
	routeBans               routeKey = "bans"
	routeBanPlayer          routeKey = "ban_player"
	routeReport             routeKey = "report"
	routeProfileSettings    routeKey = "profile_settings"
	routeAppeal             routeKey = "appeal"
	routeLogin              routeKey = "login"
	routeLogout             routeKey = "logout"
	routeLoginCallback      routeKey = "login_callback"
	routeLoginSuccess       routeKey = "login_success"
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
	router.Use(gin.Logger())
	// Dont use session for static assets
	router.StaticFS("/dist", http.Dir(config.HTTP.StaticPath))

	tokenAuthed := router.Use(authMiddleWare())

	tokenAuthed.GET(routeRaw(string(routeHome)), onIndex())
	tokenAuthed.GET(routeRaw(string(routeLoginCallback)), onOpenIDCallback())
	tokenAuthed.GET(routeRaw(string(routeLogin)), onGetLogin())
	tokenAuthed.GET(routeRaw(string(routeLogout)), onGetLogout())
	tokenAuthed.GET(routeRaw(string(routeLoginSuccess)), onLoginSuccess())

	// Client API
	tokenAuthed.GET(routeRaw(string(routeAPIServers)), onAPIGetServers())
	tokenAuthed.POST(routeRaw(string(routeAppeal)), onAPIPostAppeal())
	tokenAuthed.GET(routeRaw(string(routeAPIStats)), onAPIGetStats())
	tokenAuthed.GET(routeRaw(string(routeAPIBan)), onAPIGetBans())
	tokenAuthed.GET(routeRaw(string(routeAPIFilteredWords)), onAPIGetFilteredWords())
	tokenAuthed.GET(routeRaw(string(routeAPIProfile)), onAPIProfile())
	tokenAuthed.POST(routeRaw(string(routeAPIBan)), onAPIPostBan())

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
		routeHome:               "/",
		routeDist:               "/dist",
		routeAPIServers:         "/servers",
		routeBans:               "/bans",
		routeBanPlayer:          "/ban",
		routeReport:             "/report",
		routeAppeal:             "/appeal",
		routeLogin:              "/auth/login",
		routeLoginCallback:      "/auth/callback",
		routeLogout:             "/auth/logout",
		routeProfileSettings:    "/profile/settings",
		routeLoginSuccess:       "/login/success",
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
