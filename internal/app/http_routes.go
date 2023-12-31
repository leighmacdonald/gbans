package app

import (
	"context"
	"net/http"
	"path/filepath"
	"runtime"

	"github.com/Depado/ginprom"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/consts"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/unrolled/secure"
	"github.com/unrolled/secure/cspbuilder"
	"go.uber.org/zap"
)

func prometheusHandler() gin.HandlerFunc {
	h := promhttp.Handler()

	return func(ctx *gin.Context) {
		h.ServeHTTP(ctx.Writer, ctx.Request)
	}
}

func httpErrorHandler(logger *zap.Logger) gin.HandlerFunc {
	log := logger.Named(runtime.FuncForPC(make([]uintptr, 10)[0]).Name())

	return func(c *gin.Context) {
		c.Next()

		for _, ginErr := range c.Errors {
			log.Error("Unhandled HTTP Error", zap.Error(ginErr))
		}
	}
}

func useSecure(mode RunMode, cspOrigin string) gin.HandlerFunc {
	cspBuilder := cspbuilder.Builder{
		Directives: map[string][]string{
			cspbuilder.DefaultSrc: {"'self'", cspOrigin},
			cspbuilder.StyleSrc:   {"'self'", "'unsafe-inline'", "https://fonts.cdnfonts.com", "https://fonts.googleapis.com"},
			cspbuilder.ScriptSrc:  {"'self'", "'unsafe-inline'", "https://www.google-analytics.com"}, // TODO  "'strict-dynamic'", "$NONCE",
			cspbuilder.FontSrc:    {"'self'", "data:", "https://fonts.gstatic.com", "https://fonts.cdnfonts.com"},
			cspbuilder.ImgSrc:     append([]string{"'self'", "data:", "https://*.tile.openstreetmap.org", "https://*.steamstatic.com", "http://localhost:9000"}, cspOrigin),
			cspbuilder.BaseURI:    {"'self'"},
			cspbuilder.ObjectSrc:  {"'none'"},
		},
	}
	secureMiddleware := secure.New(secure.Options{
		FrameDeny:             true,
		ContentTypeNosniff:    true,
		ContentSecurityPolicy: cspBuilder.MustBuild(),
		IsDevelopment:         mode != ReleaseMode,
	})

	secureFunc := func(ctx *gin.Context) {
		err := secureMiddleware.Process(ctx.Writer, ctx.Request)
		if err != nil {
			ctx.Abort()

			return
		}

		// Avoid header rewrite if response is a redirection.
		if status := ctx.Writer.Status(); status > 300 && status < 399 {
			ctx.Abort()
		}
	}

	return secureFunc
}

// jsConfig contains all the variables that we inject into the frontend at runtime.
type jsConfig struct {
	SiteName        string `json:"site_name"`
	DiscordClientID string `json:"discord_client_id"`
	DiscordLinkID   string `json:"discord_link_id"`
	// External URL used to access S3 assets. media:// links are replaces with this url
	AssetURL     string `json:"asset_url"`
	BucketDemo   string `json:"bucket_demo"`
	BucketMedia  string `json:"bucket_media"`
	BuildVersion string `json:"build_version"`
	BuildCommit  string `json:"build_commit"`
	BuildDate    string `json:"build_date"`
}

//nolint:contextcheck,maintidx
func createRouter(ctx context.Context, app *App) *gin.Engine {
	engine := gin.New()
	engine.MaxMultipartMemory = 8 << 24

	if app.conf.General.Mode != ReleaseMode {
		pprof.Register(engine)
	}

	if app.conf.General.Mode != TestMode {
		engine.Use(httpErrorHandler(app.log), gin.Recovery())
		engine.Use(useSecure(app.conf.General.Mode, app.conf.S3.ExternalURL))

		corsConfig := cors.DefaultConfig()
		corsConfig.AllowOrigins = app.conf.HTTP.CorsOrigins
		corsConfig.AllowHeaders = append(corsConfig.AllowHeaders, "Authorization")
		corsConfig.AllowWildcard = false
		corsConfig.AllowCredentials = false
		engine.Use(cors.New(corsConfig))
	}

	prom := ginprom.New(func(prom *ginprom.Prometheus) {
		prom.Namespace = "gbans"
		prom.Subsystem = "http"
	})
	engine.Use(prom.Instrument())

	staticPath := app.conf.HTTP.StaticPath
	if staticPath == "" {
		staticPath = "./dist"
	}

	absStaticPath, errStaticPath := filepath.Abs(staticPath)
	if errStaticPath != nil {
		app.log.Fatal("Invalid static path", zap.Error(errStaticPath))
	}

	engine.StaticFS("/dist", http.Dir(absStaticPath))

	if app.conf.General.Mode != TestMode {
		engine.LoadHTMLFiles(filepath.Join(absStaticPath, "index.html"))
	}

	// These should match  defined in the frontend. This allows us to use the browser
	// based routing when serving the SPA.
	jsRoutes := []string{
		"/", "/servers", "/profile/:steam_id", "/bans", "/appeal", "/settings", "/report",
		"/admin/server_logs", "/admin/servers", "/admin/people", "/admin/ban/steam", "/admin/ban/cidr",
		"/admin/ban/asn", "/admin/ban/group", "/admin/reports", "/admin/news", "/admin/import", "/admin/filters",
		"/404", "/logout", "/login/success", "/report/:report_id", "/wiki", "/wiki/*slug", "/log/:match_id",
		"/logs/:steam_id", "/logs", "/ban/:ban_id", "/chatlogs", "/admin/appeals", "/login", "/pug", "/quickplay",
		"/global_stats", "/stv", "/login/discord", "/notifications", "/admin/network", "/stats",
		"/stats/weapon/:weapon_id", "/stats/player/:steam_id", "/privacy-policy", "/admin/contests",
		"/contests", "/contests/:contest_id", "/forums", "/forums/:forum_id", "/forums/thread/:forum_thread_id",
	}
	for _, rt := range jsRoutes {
		engine.GET(rt, func(c *gin.Context) {
			c.HTML(http.StatusOK, "index.html", jsConfig{
				SiteName:        app.conf.General.SiteName,
				DiscordClientID: app.conf.Discord.AppID,
				DiscordLinkID:   app.conf.Discord.LinkID,
				AssetURL:        app.conf.S3.ExternalURL,
				BucketDemo:      app.conf.S3.BucketDemo,
				BucketMedia:     app.conf.S3.BucketMedia,
				BuildVersion:    BuildVersion,
				BuildCommit:     BuildCommit,
				BuildDate:       BuildDate,
			})
		})
	}

	engine.GET("/auth/callback", onOpenIDCallback(app))
	engine.GET("/export/bans/tf2bd", onAPIExportBansTF2BD(app))
	engine.GET("/export/bans/valve/steamid", onAPIExportBansValveSteamID(app))
	engine.GET("/metrics", prometheusHandler())

	engine.GET("/api/profile", onAPIProfile(app))
	engine.GET("/api/servers/state", onAPIGetServerStates(app))
	engine.GET("/api/stats", onAPIGetStats(app))

	engine.POST("/api/news_latest", onAPIGetNewsLatest(app))

	engine.GET("/api/patreon/campaigns", onAPIGetPatreonCampaigns(app))

	engine.GET("/media/:media_id", onGetMediaByID(app))
	engine.GET("/api/servers", onAPIGetServers(app))

	engine.GET("/api/stats/map", onAPIGetMapUsage(app))
	engine.POST("/api/demos", onAPIPostDemosQuery(app))

	// Service discovery endpoints
	engine.GET("/api/sd/prometheus/hosts", onAPIGetPrometheusHosts(app))
	engine.GET("/api/sd/ansible/hosts", onAPIGetPrometheusHosts(app))

	// Game server plugin routes
	engine.POST("/api/server/auth", onSAPIPostServerAuth(app))

	engine.GET("/export/sourcemod/admins_simple.ini", onAPIExportSourcemodSimpleAdmins(app))

	engine.GET("/api/forum/active_users", onAPIActiveUsers(app))

	engine.POST("/api/auth/refresh", onTokenRefresh(app))

	// This allows use of the user profile on endpoints that have optional authentication
	optionalAuth := engine.Group("/")
	{
		optional := optionalAuth.Use(authMiddleware(app, consts.PGuest))
		optional.GET("/api/contests", onAPIGetContests(app))
		optional.GET("/api/contests/:contest_id", onAPIGetContest(app))
		optional.GET("/api/contests/:contest_id/entries", onAPIGetContestEntries(app))
		optional.GET("/api/forum/overview", onAPIForumOverview(app))
		optional.GET("/api/forum/messages/recent", onAPIForumMessagesRecent(app))
		optional.POST("/api/forum/threads", onAPIForumThreads(app))
		optional.GET("/api/forum/thread/:forum_thread_id", onAPIForumThread(app))
		optional.GET("/api/wiki/slug/*slug", onAPIGetWikiSlug(app))
		optional.GET("/api/forum/forum/:forum_id", onAPIForum(app))
		optional.POST("/api/forum/messages", onAPIForumMessages(app))
	}

	srvGrp := engine.Group("/")
	{
		// Server Auth Request
		serverAuth := srvGrp.Use(authServerMiddleWare(app))
		serverAuth.GET("/api/server/admins", onAPIGetServerAdmins(app))
		serverAuth.POST("/api/ping_mod", onAPIPostPingMod(app))
		serverAuth.POST("/api/check", onAPIPostServerCheck(app))
		serverAuth.POST("/api/demo", onAPIPostDemo(app))
		// Duplicated since we need to authenticate via server middleware
		serverAuth.POST("/api/sm/bans/steam/create", onAPIPostBanSteamCreate(app))
		serverAuth.POST("/api/sm/report/create", onAPIPostReportCreate(app))
		serverAuth.POST("/api/state_update", onAPIPostServerState(app))
	}

	authedGrp := engine.Group("/")
	{
		// Basic logged-in user
		authed := authedGrp.Use(authMiddleware(app, consts.PUser))

		authed.GET("/api/auth/discord", onOAuthDiscordCallback(app))
		authed.GET("/api/auth/logout", onAPILogout(app))
		authed.GET("/api/current_profile", onAPICurrentProfile(app))
		authed.POST("/api/current_profile/notifications", onAPICurrentProfileNotifications(app))

		authed.GET("/api/current_profile/settings", onAPIGetPersonSettings(app))
		authed.POST("/api/current_profile/settings", onAPIPostPersonSettings(app))

		authed.POST("/api/report", onAPIPostReportCreate(app))
		authed.GET("/api/report/:report_id", onAPIGetReport(app))
		authed.POST("/api/reports", onAPIGetReports(app))
		authed.POST("/api/report_status/:report_id", onAPISetReportStatus(app))
		authed.POST("/api/media", onAPISaveMedia(app))

		authed.GET("/api/report/:report_id/messages", onAPIGetReportMessages(app))
		authed.POST("/api/report/:report_id/messages", onAPIPostReportMessage(app))
		authed.POST("/api/report/message/:report_message_id", onAPIEditReportMessage(app))
		authed.DELETE("/api/report/message/:report_message_id", onAPIDeleteReportMessage(app))
		authed.GET("/api/bans/steam/:ban_id", onAPIGetBanByID(app))
		authed.GET("/api/bans/:ban_id/messages", onAPIGetBanMessages(app))
		authed.POST("/api/bans/:ban_id/messages", onAPIPostBanMessage(app))
		authed.POST("/api/bans/message/:ban_message_id", onAPIEditBanMessage(app))
		authed.DELETE("/api/bans/message/:ban_message_id", onAPIDeleteBanMessage(app))
		authed.GET("/api/sourcebans/:steam_id", onAPIGetSourceBans(app))

		authed.GET("/api/log/:match_id", onAPIGetMatch(app))
		authed.POST("/api/logs", onAPIGetMatches(app))
		authed.POST("/api/messages", onAPIQueryMessages(app))

		authed.GET("/api/stats/weapons", onAPIGetStatsWeaponsOverall(ctx, app))
		authed.GET("/api/stats/weapon/:weapon_id", onAPIGetsStatsWeapon(app))
		authed.GET("/api/stats/players", onAPIGetStatsPlayersOverall(ctx, app))
		authed.GET("/api/stats/healers", onAPIGetStatsHealersOverall(ctx, app))
		authed.GET("/api/stats/player/:steam_id/weapons", onAPIGetPlayerWeaponStatsOverall(app))
		authed.GET("/api/stats/player/:steam_id/classes", onAPIGetPlayerClassStatsOverall(app))
		authed.GET("/api/stats/player/:steam_id/overall", onAPIGetPlayerStatsOverall(app))

		authed.POST("/api/contests/:contest_id/upload", onAPISaveContestEntryMedia(app))
		authed.GET("/api/contests/:contest_id/vote/:contest_entry_id/:direction", onAPISaveContestEntryVote(app))
		authed.POST("/api/contests/:contest_id/submit", onAPISaveContestEntrySubmit(app))
		authed.DELETE("/api/contest_entry/:contest_entry_id", onAPIDeleteContestEntry(app))

		authed.POST("/api/forum/forum/:forum_id/thread", onAPIThreadCreate(app))
		authed.POST("/api/forum/thread/:forum_thread_id/message", onAPIThreadCreateReply(app))
		authed.POST("/api/forum/message/:forum_message_id", onAPIThreadMessageUpdate(app))
		authed.DELETE("/api/forum/thread/:forum_thread_id", onAPIThreadDelete(app))
		authed.DELETE("/api/forum/message/:forum_message_id", onAPIMessageDelete(app))
		authed.POST("/api/forum/thread/:forum_thread_id", onAPIThreadUpdate(app))
	}

	editorGrp := engine.Group("/")
	{
		// Editor access
		editorRoute := editorGrp.Use(authMiddleware(app, consts.PEditor))
		editorRoute.POST("/api/wiki/slug", onAPISaveWikiSlug(app))
		editorRoute.POST("/api/news", onAPIPostNewsCreate(app))
		editorRoute.POST("/api/news/:news_id", onAPIPostNewsUpdate(app))
		editorRoute.POST("/api/news_all", onAPIGetNewsAll(app))
		editorRoute.POST("/api/filters/query", onAPIQueryWordFilters(app))
		editorRoute.POST("/api/filters", onAPIPostWordFilter(app))
		editorRoute.DELETE("/api/filters/:word_id", onAPIDeleteWordFilter(app))
		editorRoute.POST("/api/filter_match", onAPIPostWordMatch(app))
		editorRoute.GET("/export/bans/valve/network", onAPIExportBansValveIP(app))
		editorRoute.POST("/api/players", onAPISearchPlayers(app))
	}

	modGrp := engine.Group("/")
	{
		// Moderator access
		modRoute := modGrp.Use(authMiddleware(app, consts.PModerator))
		modRoute.POST("/api/report/:report_id/state", onAPIPostBanState(app))
		modRoute.POST("/api/connections", onAPIQueryPersonConnections(app))
		modRoute.GET("/api/message/:person_message_id/context/:padding", onAPIQueryMessageContext(app))
		modRoute.POST("/api/appeals", onAPIGetAppeals(app))

		modRoute.POST("/api/bans/steam", onAPIGetBansSteam(app))
		modRoute.POST("/api/bans/steam/create", onAPIPostBanSteamCreate(app))
		modRoute.DELETE("/api/bans/steam/:ban_id", onAPIPostBanDelete(app))
		modRoute.POST("/api/bans/steam/:ban_id", onAPIPostBanUpdate(app))
		modRoute.POST("/api/bans/steam/:ban_id/status", onAPIPostSetBanAppealStatus(app))

		modRoute.POST("/api/bans/cidr/create", onAPIPostBansCIDRCreate(app))
		modRoute.POST("/api/bans/cidr", onAPIGetBansCIDR(app))
		modRoute.DELETE("/api/bans/cidr/:net_id", onAPIDeleteBansCIDR(app))
		modRoute.POST("/api/bans/cidr/:net_id", onAPIPostBansCIDRUpdate(app))

		modRoute.POST("/api/bans/asn/create", onAPIPostBansASNCreate(app))
		modRoute.POST("/api/bans/asn", onAPIGetBansASN(app))
		modRoute.DELETE("/api/bans/asn/:asn_id", onAPIDeleteBansASN(app))
		modRoute.POST("/api/bans/asn/:asn_id", onAPIPostBansASNUpdate(app))

		modRoute.POST("/api/bans/group/create", onAPIPostBansGroupCreate(app))
		modRoute.POST("/api/bans/group", onAPIGetBansGroup(app))
		modRoute.DELETE("/api/bans/group/:ban_group_id", onAPIDeleteBansGroup(app))
		modRoute.POST("/api/bans/group/:ban_group_id", onAPIPostBansGroupUpdate(app))

		modRoute.GET("/api/patreon/pledges", onAPIGetPatreonPledges(app))

		modRoute.POST("/api/contests", onAPIPostContest(app))
		modRoute.DELETE("/api/contests/:contest_id", onAPIDeleteContest(app))
		modRoute.PUT("/api/contests/:contest_id", onAPIUpdateContest(app))

		modRoute.POST("/api/forum/category", onAPICreateForumCategory(app))
		modRoute.GET("/api/forum/category/:forum_category_id", onAPIForumCategory(app))
		modRoute.POST("/api/forum/category/:forum_category_id", onAPIUpdateForumCategory(app))
		modRoute.POST("/api/forum/forum", onAPICreateForumForum(app))
		modRoute.POST("/api/forum/forum/:forum_id", onAPIUpdateForumForum(app))

		modRoute.POST("/api/block_list/whitelist", onAPIPostBlockListWhitelistCreate(app))
		modRoute.POST("/api/block_list/whitelist/:cidr_block_whitelist_id", onAPIPostBlockListWhitelistUpdate(app))
		modRoute.DELETE("/api/block_list/whitelist/:cidr_block_whitelist_id", onAPIDeleteBlockListWhitelist(app))
		modRoute.GET("/api/block_list", onAPIGetBlockLists(app))
		modRoute.POST("/api/block_list/checker", onAPIPostBlocklistCheck(app))
	}

	adminGrp := engine.Group("/")
	{
		// Admin access
		adminRoute := adminGrp.Use(authMiddleware(app, consts.PAdmin))
		adminRoute.POST("/api/servers", onAPIPostServer(app))
		adminRoute.POST("/api/servers/:server_id", onAPIPostServerUpdate(app))
		adminRoute.DELETE("/api/servers/:server_id", onAPIPostServerDelete(app))
		adminRoute.POST("/api/servers_admin", onAPIGetServersAdmin(app))
		adminRoute.PUT("/api/player/:steam_id/permissions", onAPIPutPlayerPermission(app))

		adminRoute.POST("/api/block_list", onAPIPostBlockListCreate(app))
		adminRoute.POST("/api/block_list/:cidr_block_source_id", onAPIPostBlockListUpdate(app))
		adminRoute.DELETE("/api/block_list/:cidr_block_source_id", onAPIDeleteBlockList(app))
	}

	return engine
}
