package app

import (
	"github.com/Depado/ginprom"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/model"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
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

func ErrorHandler(logger *log.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		for _, ginErr := range c.Errors {
			logger.Error(ginErr)
		}
	}
}

func (web *web) setupRouter(engine *gin.Engine) {
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowOrigins = config.HTTP.CorsOrigins
	corsConfig.AllowHeaders = []string{"*"}
	corsConfig.AllowWildcard = false
	corsConfig.AllowCredentials = false
	corsConfig.AddAllowMethods("OPTIONS")
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
		log.Fatalf("Invalid static path: %v", errStaticPath)
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
		"/pug", "/quickplay", "/global_stats", "/stv", "/login/discord"}
	for _, rt := range jsRoutes {
		engine.GET(rt, func(c *gin.Context) {
			c.HTML(http.StatusOK, "index.html", gin.H{
				"jsGlobals": gin.H{
					"siteName":        config.General.SiteName,
					"discordClientId": config.Discord.AppID,
				},
				"siteName": config.General.SiteName,
			})
		})
	}
	engine.GET("/auth/callback", web.onOpenIDCallback())
	engine.GET("/api/auth/logout", web.onGetLogout())
	engine.POST("/api/auth/refresh", web.onTokenRefresh())

	engine.GET("/export/bans/tf2bd", web.onAPIExportBansTF2BD())
	engine.GET("/export/sourcemod/admins_simple.ini", web.onAPIExportSourcemodSimpleAdmins())
	engine.GET("/export/bans/valve/steamid", web.onAPIExportBansValveSteamId())
	engine.GET("/export/bans/valve/network", web.onAPIExportBansValveIP())
	engine.GET("/metrics", prometheusHandler())

	engine.GET("/api/profile", web.onAPIProfile())
	engine.GET("/api/servers/state", web.onAPIGetServerStates())
	engine.GET("/api/stats", web.onAPIGetStats())
	engine.GET("/api/competitive", web.onAPIGetCompHist())

	engine.GET("/api/players", web.onAPIGetPlayers())
	engine.GET("/api/wiki/slug/*slug", web.onAPIGetWikiSlug())
	engine.GET("/api/log/:match_id", web.onAPIGetMatch())
	engine.POST("/api/logs", web.onAPIGetMatches())
	engine.GET("/media/:media_id", web.onGetMediaById())
	engine.POST("/api/news_latest", web.onAPIGetNewsLatest())
	engine.POST("/api/server_query", web.onAPIPostServerQuery())
	engine.GET("/api/server_stats", web.onAPIGetTF2Stats())

	engine.POST("/api/demos", web.onAPIPostDemosQuery())
	engine.GET("/demos/name/:demo_name", web.onAPIGetDemoDownloadByName())
	engine.GET("/demos/:demo_id", web.onAPIGetDemoDownload())

	// Service discovery endpoints
	engine.GET("/api/sd/prometheus/hosts", web.onAPIGetPrometheusHosts())
	engine.GET("/api/sd/ansible/hosts", web.onAPIGetPrometheusHosts())

	// Game server plugin routes
	engine.POST("/api/server/auth", web.onSAPIPostServerAuth())
	engine.POST("/api/resolve_profile", web.onAPIGetResolveProfile())

	engine.GET("/api/patreon/campaigns", web.onAPIGetPatreonCampaigns())
	engine.GET("/api/patreon/pledges", web.onAPIGetPatreonPledges())

	srvGrp := engine.Group("/")
	{
		// Server Auth Request
		serverAuth := srvGrp.Use(web.authServerMiddleWare())
		serverAuth.GET("/api/server/admins", web.onAPIGetServerAdmins())
		serverAuth.POST("/api/ping_mod", web.onAPIPostPingMod())
		serverAuth.POST("/api/check", web.onAPIPostServerCheck())
		serverAuth.POST("/api/demo", web.onAPIPostDemo())
		serverAuth.POST("/api/log", web.onAPIPostLog())
		// Duplicated since we need to authenticate via server middleware
		serverAuth.POST("/api/sm/bans/steam/create", web.onAPIPostBanSteamCreate())
		serverAuth.POST("/api/sm/report/create", web.onAPIPostReportCreate())
	}

	authedGrp := engine.Group("/")
	{
		// Basic logged-in user
		authed := authedGrp.Use(authMiddleware(web.app.store, model.PUser))
		authed.GET("/ws", func(c *gin.Context) {
			web.wsConnHandler(c.Writer, c.Request, currentUserProfile(c))
		})

		authed.GET("/api/auth/discord", web.onOAuthDiscordCallback())
		authed.GET("/api/current_profile", web.onAPICurrentProfile())
		authed.GET("/api/current_profile/notifications", web.onAPICurrentProfileNotifications())
		authed.POST("/api/report", web.onAPIPostReportCreate())
		authed.GET("/api/report/:report_id", web.onAPIGetReport())
		authed.POST("/api/reports", web.onAPIGetReports())
		authed.POST("/api/report_status/:report_id", web.onAPISetReportStatus())
		authed.POST("/api/media", web.onAPISaveMedia())

		authed.GET("/api/report/:report_id/messages", web.onAPIGetReportMessages())
		authed.POST("/api/report/:report_id/messages", web.onAPIPostReportMessage())
		authed.POST("/api/report/message/:report_message_id", web.onAPIEditReportMessage())
		authed.DELETE("/api/report/message/:report_message_id", web.onAPIDeleteReportMessage())

		authed.GET("/api/bans/steam/:ban_id", web.onAPIGetBanByID())
		authed.GET("/api/bans/:ban_id/messages", web.onAPIGetBanMessages())
		authed.POST("/api/bans/:ban_id/messages", web.onAPIPostBanMessage())
		authed.POST("/api/bans/message/:ban_message_id", web.onAPIEditBanMessage())
		authed.DELETE("/api/bans/message/:ban_message_id", web.onAPIDeleteBanMessage())
	}

	editorGrp := engine.Group("/")
	{
		// Editor access
		editorRoute := editorGrp.Use(authMiddleware(web.app.store, model.PEditor))
		editorRoute.POST("/api/wiki/slug", web.onAPISaveWikiSlug())
		editorRoute.POST("/api/news", web.onAPIPostNewsCreate())
		editorRoute.POST("/api/news/:news_id", web.onAPIPostNewsUpdate())
		editorRoute.POST("/api/news_all", web.onAPIGetNewsAll())
		editorRoute.GET("/api/filters", web.onAPIGetWordFilters())
		editorRoute.POST("/api/filters", web.onAPIPostWordFilter())
		editorRoute.DELETE("/api/filters/:word_id", web.onAPIDeleteWordFilter())
		editorRoute.POST("/api/filter_match", web.onAPIPostWordMatch())
	}

	modGrp := engine.Group("/")
	{
		// Moderator access
		modRoute := modGrp.Use(authMiddleware(web.app.store, model.PModerator))
		modRoute.POST("/api/report/:report_id/state", web.onAPIPostBanState())
		modRoute.GET("/api/connections/:steam_id", web.onAPIGetPersonConnections())
		modRoute.GET("/api/messages/:steam_id", web.onAPIGetPersonMessages())
		modRoute.GET("/api/message/:person_message_id/context", web.onAPIGetMessageContext())
		modRoute.POST("/api/messages", web.onAPIQueryMessages())
		modRoute.POST("/api/appeals", web.onAPIGetAppeals())
		modRoute.POST("/api/bans/steam", web.onAPIGetBansSteam())
		modRoute.POST("/api/bans/steam/create", web.onAPIPostBanSteamCreate())
		modRoute.DELETE("/api/bans/steam/:ban_id", web.onAPIPostBanDelete())
		modRoute.POST("/api/bans/steam/:ban_id/status", web.onAPIPostSetBanAppealStatus())
		modRoute.POST("/api/bans/cidr/create", web.onAPIPostBansCIDRCreate())
		modRoute.POST("/api/bans/cidr", web.onAPIGetBansCIDR())
		modRoute.DELETE("/api/bans/cidr/:net_id", web.onAPIDeleteBansCIDR())
		modRoute.POST("/api/bans/asn/create", web.onAPIPostBansASNCreate())
		modRoute.POST("/api/bans/asn", web.onAPIGetBansASN())
		modRoute.DELETE("/api/bans/asn/:asn_id", web.onAPIDeleteBansASN())
		modRoute.POST("/api/bans/group/create", web.onAPIPostBansGroupCreate())
		modRoute.POST("/api/bans/group", web.onAPIGetBansGroup())
		modRoute.DELETE("/api/bans/group/:ban_group_id", web.onAPIDeleteBansGroup())
	}

	adminGrp := engine.Group("/")
	{
		// Admin access
		adminRoute := adminGrp.Use(authMiddleware(web.app.store, model.PAdmin))
		adminRoute.POST("/api/servers", web.onAPIPostServer())
		adminRoute.POST("/api/servers/:server_id", web.onAPIPostServerUpdate())
		adminRoute.DELETE("/api/servers/:server_id", web.onAPIPostServerDelete())
		adminRoute.GET("/api/servers", web.onAPIGetServers())
	}
}
