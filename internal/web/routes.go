package web

import (
	"github.com/Depado/ginprom"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"net/http"
	"path/filepath"
)

func prometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()
	return func(ctx *gin.Context) {
		h.ServeHTTP(ctx.Writer, ctx.Request)
	}
}

var registered = false

func ErrorHandler(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		for _, ginErr := range c.Errors {
			logger.Error("Unhandled HTTP Error", zap.Error(ginErr))
		}
	}
}

// jsConfig contains all the variables that we inject into the frontend at runtime
type jsConfig struct {
	SiteName        string `json:"siteName"`
	DiscordClientId string `json:"discordClientId"`
	DiscordLinkId   string `json:"discordLinkId"`
}

func createRouter() *gin.Engine {
	engine := gin.New()
	engine.Use(ErrorHandler(logger), gin.Recovery())

	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = config.HTTP.CorsOrigins
	corsConfig.AllowHeaders = []string{"*"}
	corsConfig.AllowWildcard = true
	corsConfig.AllowCredentials = false
	if config.General.Mode != config.TestMode {
		engine.Use(cors.New(corsConfig))
	}
	if !registered {
		prom := ginprom.New(func(prom *ginprom.Prometheus) {
			prom.Namespace = "gbans"
			prom.Subsystem = "http"
		})
		engine.Use(prom.Instrument())
		registered = true
	}
	staticPath := config.HTTP.StaticPath
	if staticPath == "" {
		staticPath = "./dist"
	}
	absStaticPath, errStaticPath := filepath.Abs(staticPath)
	if errStaticPath != nil {
		logger.Fatal("Invalid static path", zap.Error(errStaticPath))
	}

	engine.StaticFS("/dist", http.Dir(absStaticPath))
	engine.LoadHTMLFiles(filepath.Join(absStaticPath, "index.html"))

	// These should match routes defined in the frontend. This allows us to use the browser
	// based routing when serving the SPA.
	jsRoutes := []string{
		"/", "/servers", "/profile/:steam_id", "/bans", "/appeal", "/settings", "/report",
		"/admin/server_logs", "/admin/servers", "/admin/people", "/admin/ban", "/admin/reports", "/admin/news",
		"/admin/import", "/admin/filters", "/404", "/logout", "/login/success", "/report/:report_id", "/wiki",
		"/wiki/*slug", "/log/:match_id", "/logs", "/ban/:ban_id", "/admin/chat", "/admin/appeals", "/login",
		"/pug", "/quickplay", "/global_stats", "/stv", "/login/discord", "/notifications"}
	for _, rt := range jsRoutes {
		engine.GET(rt, func(c *gin.Context) {
			c.HTML(http.StatusOK, "index.html", jsConfig{
				SiteName:        config.General.SiteName,
				DiscordClientId: config.Discord.AppID,
				DiscordLinkId:   config.Discord.LinkId,
			})
		})
	}
	engine.GET("/auth/callback", onOpenIDCallback())
	engine.GET("/api/auth/logout", onGetLogout())
	engine.POST("/api/auth/refresh", onTokenRefresh())

	engine.GET("/export/bans/tf2bd", onAPIExportBansTF2BD())
	engine.GET("/export/sourcemod/admins_simple.ini", onAPIExportSourcemodSimpleAdmins())
	engine.GET("/export/bans/valve/steamid", onAPIExportBansValveSteamId())
	engine.GET("/export/bans/valve/network", onAPIExportBansValveIP())
	engine.GET("/metrics", prometheusHandler())

	engine.GET("/api/profile", onAPIProfile())
	engine.GET("/api/servers/state", onAPIGetServerStates())
	engine.GET("/api/stats", onAPIGetStats())
	engine.GET("/api/competitive", onAPIGetCompHist())

	engine.GET("/api/players", onAPIGetPlayers())
	engine.GET("/api/wiki/slug/*slug", onAPIGetWikiSlug())
	engine.GET("/api/log/:match_id", onAPIGetMatch())
	engine.POST("/api/logs", onAPIGetMatches())
	engine.GET("/media/:media_id", onGetMediaById())
	engine.POST("/api/news_latest", onAPIGetNewsLatest())
	engine.POST("/api/server_query", onAPIPostServerQuery())
	engine.GET("/api/server_stats", onAPIGetTF2Stats())

	engine.POST("/api/demos", onAPIPostDemosQuery())
	engine.GET("/demos/name/:demo_name", onAPIGetDemoDownloadByName())
	engine.GET("/demos/:demo_id", onAPIGetDemoDownload())

	// Service discovery endpoints
	engine.GET("/api/sd/prometheus/hosts", onAPIGetPrometheusHosts())
	engine.GET("/api/sd/ansible/hosts", onAPIGetPrometheusHosts())

	// Game server plugin routes
	engine.POST("/api/server/auth", onSAPIPostServerAuth())
	engine.POST("/api/resolve_profile", onAPIGetResolveProfile())

	engine.GET("/api/patreon/campaigns", onAPIGetPatreonCampaigns())
	engine.GET("/api/patreon/pledges", onAPIGetPatreonPledges())

	srvGrp := engine.Group("/")
	{
		// Server Auth Request
		serverAuth := srvGrp.Use(authServerMiddleWare())
		serverAuth.GET("/api/server/admins", onAPIGetServerAdmins())
		serverAuth.POST("/api/ping_mod", onAPIPostPingMod())
		serverAuth.POST("/api/check", onAPIPostServerCheck())
		serverAuth.POST("/api/demo", onAPIPostDemo())
		serverAuth.POST("/api/log", onAPIPostLog())
		// Duplicated since we need to authenticate via server middleware
		serverAuth.POST("/api/sm/bans/steam/create", onAPIPostBanSteamCreate())
		serverAuth.POST("/api/sm/report/create", onAPIPostReportCreate())
	}

	authedGrp := engine.Group("/")
	{
		// Basic logged-in user
		authed := authedGrp.Use(authMiddleware(consts.PUser))
		authed.GET("/ws", func(c *gin.Context) {
			wsConnHandler(c.Writer, c.Request, currentUserProfile(c))
		})

		authed.GET("/api/auth/discord", onOAuthDiscordCallback())
		authed.GET("/api/current_profile", onAPICurrentProfile())
		authed.POST("/api/current_profile/notifications", onAPICurrentProfileNotifications())
		authed.POST("/api/report", onAPIPostReportCreate())
		authed.GET("/api/report/:report_id", onAPIGetReport())
		authed.POST("/api/reports", onAPIGetReports())
		authed.POST("/api/report_status/:report_id", onAPISetReportStatus())
		authed.POST("/api/media", onAPISaveMedia())

		authed.GET("/api/report/:report_id/messages", onAPIGetReportMessages())
		authed.POST("/api/report/:report_id/messages", onAPIPostReportMessage())
		authed.POST("/api/report/message/:report_message_id", onAPIEditReportMessage())
		authed.DELETE("/api/report/message/:report_message_id", onAPIDeleteReportMessage())

		authed.GET("/api/bans/steam/:ban_id", onAPIGetBanByID())
		authed.GET("/api/bans/:ban_id/messages", onAPIGetBanMessages())
		authed.POST("/api/bans/:ban_id/messages", onAPIPostBanMessage())
		authed.POST("/api/bans/message/:ban_message_id", onAPIEditBanMessage())
		authed.DELETE("/api/bans/message/:ban_message_id", onAPIDeleteBanMessage())
	}

	editorGrp := engine.Group("/")
	{
		// Editor access
		editorRoute := editorGrp.Use(authMiddleware(consts.PEditor))
		editorRoute.POST("/api/wiki/slug", onAPISaveWikiSlug())
		editorRoute.POST("/api/news", onAPIPostNewsCreate())
		editorRoute.POST("/api/news/:news_id", onAPIPostNewsUpdate())
		editorRoute.POST("/api/news_all", onAPIGetNewsAll())
		editorRoute.GET("/api/filters", onAPIGetWordFilters())
		editorRoute.POST("/api/filters", onAPIPostWordFilter())
		editorRoute.DELETE("/api/filters/:word_id", onAPIDeleteWordFilter())
		editorRoute.POST("/api/filter_match", onAPIPostWordMatch())
	}

	modGrp := engine.Group("/")
	{
		// Moderator access
		modRoute := modGrp.Use(authMiddleware(consts.PModerator))
		modRoute.POST("/api/report/:report_id/state", onAPIPostBanState())
		modRoute.GET("/api/connections/:steam_id", onAPIGetPersonConnections())
		modRoute.GET("/api/messages/:steam_id", onAPIGetPersonMessages())
		modRoute.GET("/api/message/:person_message_id/context", onAPIGetMessageContext())
		modRoute.POST("/api/messages", onAPIQueryMessages())
		modRoute.POST("/api/appeals", onAPIGetAppeals())
		modRoute.POST("/api/bans/steam", onAPIGetBansSteam())
		modRoute.POST("/api/bans/steam/create", onAPIPostBanSteamCreate())
		modRoute.DELETE("/api/bans/steam/:ban_id", onAPIPostBanDelete())
		modRoute.POST("/api/bans/steam/:ban_id/status", onAPIPostSetBanAppealStatus())
		modRoute.POST("/api/bans/cidr/create", onAPIPostBansCIDRCreate())
		modRoute.POST("/api/bans/cidr", onAPIGetBansCIDR())
		modRoute.DELETE("/api/bans/cidr/:net_id", onAPIDeleteBansCIDR())
		modRoute.POST("/api/bans/asn/create", onAPIPostBansASNCreate())
		modRoute.POST("/api/bans/asn", onAPIGetBansASN())
		modRoute.DELETE("/api/bans/asn/:asn_id", onAPIDeleteBansASN())
		modRoute.POST("/api/bans/group/create", onAPIPostBansGroupCreate())
		modRoute.POST("/api/bans/group", onAPIGetBansGroup())
		modRoute.DELETE("/api/bans/group/:ban_group_id", onAPIDeleteBansGroup())
	}

	adminGrp := engine.Group("/")
	{
		// Admin access
		adminRoute := adminGrp.Use(authMiddleware(consts.PAdmin))
		adminRoute.POST("/api/servers", onAPIPostServer())
		adminRoute.POST("/api/servers/:server_id", onAPIPostServerUpdate())
		adminRoute.DELETE("/api/servers/:server_id", onAPIPostServerDelete())
		adminRoute.GET("/api/servers", onAPIGetServers())
	}

	return engine
}
