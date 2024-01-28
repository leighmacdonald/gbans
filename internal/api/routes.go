package api

import (
	"context"
	"github.com/leighmacdonald/gbans/internal/middleware"
	"net/http"
	"path/filepath"
	"runtime"

	"github.com/Depado/ginprom"
	sentrygin "github.com/getsentry/sentry-go/gin"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/internal/domain"
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

func useSecure(mode config.RunMode, cspOrigin string) gin.HandlerFunc {
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
		IsDevelopment:         mode != config.ReleaseMode,
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
	SentryDSN    string `json:"sentry_dsn"`
}

//nolint:contextcheck,maintidx
func createRouter(ctx context.Context, env Env) *gin.Engine {
	engine := gin.New()
	engine.MaxMultipartMemory = 8 << 24
	engine.Use(gin.Recovery())

	conf := env.Config()

	if conf.Log.SentryDSN != "" {
		engine.Use(sentrygin.New(sentrygin.Options{Repanic: true}))
		engine.Use(func(ctx *gin.Context) {
			if hub := sentrygin.GetHubFromContext(ctx); hub != nil {
				hub.Scope().SetTag("version", env.Version().BuildVersion)
			}
			ctx.Next()
		})
	}

	if conf.General.Mode != config.ReleaseMode {
		pprof.Register(engine)
	}

	if conf.General.Mode != config.TestMode {
		engine.Use(httpErrorHandler(env.Log()), gin.Recovery())
		engine.Use(useSecure(conf.General.Mode, conf.S3.ExternalURL))

		corsConfig := cors.DefaultConfig()
		corsConfig.AllowOrigins = conf.HTTP.CorsOrigins
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

	staticPath := conf.HTTP.StaticPath
	if staticPath == "" {
		staticPath = "./dist"
	}

	absStaticPath, errStaticPath := filepath.Abs(staticPath)
	if errStaticPath != nil {
		env.Log().Fatal("Invalid static path", zap.Error(errStaticPath))
	}

	engine.StaticFS("/dist", http.Dir(absStaticPath))

	if conf.General.Mode != config.TestMode {
		engine.LoadHTMLFiles(filepath.Join(absStaticPath, "index.html"))
	}

	// These should findMatch  defined in the frontend. This allows us to use the browser
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
		engine.GET(rt, func(ctx *gin.Context) {
			if conf.Log.SentryDSNWeb != "" {
				ctx.Header("Document-Policy", "js-profiling")
			}

			version := env.Version()

			ctx.HTML(http.StatusOK, "index.html", jsConfig{
				SiteName:        conf.General.SiteName,
				DiscordClientID: conf.Discord.AppID,
				DiscordLinkID:   conf.Discord.LinkID,
				AssetURL:        conf.S3.ExternalURL,
				BucketDemo:      conf.S3.BucketDemo,
				BucketMedia:     conf.S3.BucketMedia,
				BuildVersion:    version.BuildVersion,
				BuildCommit:     version.Commit,
				BuildDate:       version.Date,
				SentryDSN:       conf.Log.SentryDSNWeb,
			})
		})
	}

	//engine.GET("/auth/callback", middleware.onOpenIDCallback(env))
	//engine.GET("/export/bans/tf2bd", onAPIExportBansTF2BD(env))
	//engine.GET("/export/bans/valve/steamid", onAPIExportBansValveSteamID(env))
	engine.GET("/metrics", prometheusHandler())

	engine.GET("/api/profile", onAPIProfile(env))
	// engine.GET("/api/servers/state", onAPIGetServerStates(env))
	//engine.GET("/api/stats", onAPIGetStats(env))

	engine.POST("/api/news_latest", onAPIGetNewsLatest(env))

	engine.GET("/api/patreon/campaigns", onAPIGetPatreonCampaigns(env))

	engine.GET("/media/:media_id", onGetMediaByID(env))
	// engine.GET("/api/servers", onAPIGetServers(env))

	engine.GET("/api/stats/map", onAPIGetMapUsage(env))
	engine.POST("/api/demos", onAPIPostDemosQuery(env))

	// Service discovery endpoints
	engine.GET("/api/sd/prometheus/hosts", onAPIGetPrometheusHosts(env))
	engine.GET("/api/sd/ansible/hosts", onAPIGetPrometheusHosts(env))

	// Game server plugin routes
	// engine.POST("/api/server/auth", onSAPIPostServerAuth(env))

	engine.GET("/export/sourcemod/admins_simple.ini", onAPIExportSourcemodSimpleAdmins(env))

	engine.GET("/api/forum/active_users", onAPIActiveUsers(env))

	engine.POST("/api/auth/refresh", onTokenRefresh(env))

	// This allows use of the user profile on endpoints that have optional authentication
	optionalAuth := engine.Group("/")
	{
		optional := optionalAuth.Use(middleware.authMiddleware(env, domain.PGuest))
		// optional.GET("/api/contests", onAPIGetContests(env))
		// optional.GET("/api/contests/:contest_id", onAPIGetContest(env))
		// optional.GET("/api/contests/:contest_id/entries", onAPIGetContestEntries(env))
		// optional.GET("/api/forum/overview", onAPIForumOverview(env))
		// optional.GET("/api/forum/messages/recent", onAPIForumMessagesRecent(env))
		// optional.POST("/api/forum/threads", onAPIForumThreads(env))
		// optional.GET("/api/forum/thread/:forum_thread_id", onAPIForumThread(env))
		// optional.GET("/api/wiki/slug/*slug", onAPIGetWikiSlug(env))
		// optional.GET("/api/forum/forum/:forum_id", onAPIForum(env))
		// optional.POST("/api/forum/messages", onAPIForumMessages(env))
	}

	authedGrp := engine.Group("/")
	{
		// Basic logged-in user
		authed := authedGrp.Use(middleware.authMiddleware(env, domain.PUser))

		authed.GET("/api/auth/discord", onOAuthDiscordCallback(env))
		authed.GET("/api/auth/logout", onAPILogout(env))
		authed.POST("/api/current_profile/notifications", onAPICurrentProfileNotifications(env))

		authed.POST("/api/report", onAPIPostReportCreate(env))
		authed.GET("/api/report/:report_id", onAPIGetReport(env))
		authed.POST("/api/reports", onAPIGetReports(env))
		authed.POST("/api/report_status/:report_id", onAPISetReportStatus(env))
		authed.POST("/api/media", onAPISaveMedia(env))

		authed.GET("/api/report/:report_id/messages", onAPIGetReportMessages(env))
		authed.POST("/api/report/:report_id/messages", onAPIPostReportMessage(env))
		authed.POST("/api/report/message/:report_message_id", onAPIEditReportMessage(env))
		authed.DELETE("/api/report/message/:report_message_id", onAPIDeleteReportMessage(env))
		//authed.GET("/api/bans/steam/:ban_id", onAPIGetBanByID(env))
		//authed.GET("/api/bans/:ban_id/messages", onAPIGetBanMessages(env))
		//authed.POST("/api/bans/:ban_id/messages", onAPIPostBanMessage(env))
		//authed.POST("/api/bans/message/:ban_message_id", onAPIEditBanMessage(env))
		//authed.DELETE("/api/bans/message/:ban_message_id", onAPIDeleteBanMessage(env))
		//authed.GET("/api/sourcebans/:steam_id", onAPIGetSourceBans(env))

		//authed.GET("/api/log/:match_id", onAPIGetMatch(env))
		//authed.POST("/api/logs", onAPIGetMatches(env))
		authed.POST("/api/messages", onAPIQueryMessages(env))

		//authed.GET("/api/stats/weapons", onAPIGetStatsWeaponsOverall(ctx, env))
		//authed.GET("/api/stats/weapon/:weapon_id", onAPIGetsStatsWeapon(env))
		//authed.GET("/api/stats/players", onAPIGetStatsPlayersOverall(ctx, env))
		//authed.GET("/api/stats/healers", onAPIGetStatsHealersOverall(ctx, env))
		//authed.GET("/api/stats/player/:steam_id/weapons", onAPIGetPlayerWeaponStatsOverall(env))
		//authed.GET("/api/stats/player/:steam_id/classes", onAPIGetPlayerClassStatsOverall(env))
		//authed.GET("/api/stats/player/:steam_id/overall", onAPIGetPlayerStatsOverall(env))

		// authed.POST("/api/contests/:contest_id/upload", onAPISaveContestEntryMedia(env))
		// authed.GET("/api/contests/:contest_id/vote/:contest_entry_id/:direction", onAPISaveContestEntryVote(env))
		// authed.POST("/api/contests/:contest_id/submit", onAPISaveContestEntrySubmit(env))
		// authed.DELETE("/api/contest_entry/:contest_entry_id", onAPIDeleteContestEntry(env))

		// authed.POST("/api/forum/forum/:forum_id/thread", onAPIThreadCreate(env))
		// authed.POST("/api/forum/thread/:forum_thread_id/message", onAPIThreadCreateReply(env))
		// authed.POST("/api/forum/message/:forum_message_id", onAPIThreadMessageUpdate(env))
		// authed.DELETE("/api/forum/thread/:forum_thread_id", onAPIThreadDelete(env))
		// authed.DELETE("/api/forum/message/:forum_message_id", onAPIMessageDelete(env))
		// authed.POST("/api/forum/thread/:forum_thread_id", onAPIThreadUpdate(env))
	}

	editorGrp := engine.Group("/")
	{
		// Editor access
		editorRoute := editorGrp.Use(middleware.authMiddleware(env, domain.PEditor))
		// editorRoute.POST("/api/wiki/slug", onAPISaveWikiSlug(env))
		editorRoute.POST("/api/news", onAPIPostNewsCreate(env))
		editorRoute.POST("/api/news/:news_id", onAPIPostNewsUpdate(env))
		editorRoute.POST("/api/news_all", onAPIGetNewsAll(env))
		//editorRoute.POST("/api/filters/query", onAPIQueryWordFilters(env))
		//editorRoute.GET("/api/filters/state", onAPIGetWarningState(env))
		//editorRoute.POST("/api/filters", onAPIPostWordFilter(env))
		//editorRoute.DELETE("/api/filters/:word_id", onAPIDeleteWordFilter(env))
		//editorRoute.POST("/api/filter_match", onAPIPostWordMatch(env))
		editorRoute.GET("/export/bans/valve/network", onAPIExportBansValveIP(env))
		editorRoute.POST("/api/players", onAPISearchPlayers(env))
	}

	modGrp := engine.Group("/")
	{
		// Moderator access
		modRoute := modGrp.Use(middleware.authMiddleware(env, domain.PModerator))
		modRoute.POST("/api/report/:report_id/state", onAPIPostBanState(env))
		modRoute.POST("/api/connections", onAPIQueryPersonConnections(env))
		modRoute.GET("/api/message/:person_message_id/context/:padding", onAPIQueryMessageContext(env))
		modRoute.POST("/api/appeals", onAPIGetAppeals(env))

		//modRoute.POST("/api/bans/steam", onAPIGetBansSteam(env))
		//modRoute.POST("/api/bans/steam/create", onAPIPostBanSteamCreate(env))
		//modRoute.DELETE("/api/bans/steam/:ban_id", onAPIPostBanDelete(env))
		//modRoute.POST("/api/bans/steam/:ban_id", onAPIPostBanUpdate(env))
		//modRoute.POST("/api/bans/steam/:ban_id/status", onAPIPostSetBanAppealStatus(env))
		//
		//modRoute.POST("/api/bans/cidr/create", onAPIPostBansCIDRCreate(env))
		//modRoute.POST("/api/bans/cidr", onAPIGetBansCIDR(env))
		//modRoute.DELETE("/api/bans/cidr/:net_id", onAPIDeleteBansCIDR(env))
		//modRoute.POST("/api/bans/cidr/:net_id", onAPIPostBansCIDRUpdate(env))
		//
		//modRoute.POST("/api/bans/asn/create", onAPIPostBansASNCreate(env))
		//modRoute.POST("/api/bans/asn", onAPIGetBansASN(env))
		//modRoute.DELETE("/api/bans/asn/:asn_id", onAPIDeleteBansASN(env))
		//modRoute.POST("/api/bans/asn/:asn_id", onAPIPostBansASNUpdate(env))

		modRoute.POST("/api/bans/group/create", onAPIPostBansGroupCreate(env))
		modRoute.POST("/api/bans/group", onAPIGetBansGroup(env))
		modRoute.DELETE("/api/bans/group/:ban_group_id", onAPIDeleteBansGroup(env))
		modRoute.POST("/api/bans/group/:ban_group_id", onAPIPostBansGroupUpdate(env))

		modRoute.GET("/api/patreon/pledges", onAPIGetPatreonPledges(env))

		// modRoute.POST("/api/contests", onAPIPostContest(env))
		// modRoute.DELETE("/api/contests/:contest_id", onAPIDeleteContest(env))
		// modRoute.PUT("/api/contests/:contest_id", onAPIUpdateContest(env))

		// modRoute.POST("/api/forum/category", onAPICreateForumCategory(env))
		// modRoute.GET("/api/forum/category/:forum_category_id", onAPIForumCategory(env))
		// modRoute.POST("/api/forum/category/:forum_category_id", onAPIUpdateForumCategory(env))
		// modRoute.POST("/api/forum/forum", onAPICreateForumForum(env))
		// modRoute.POST("/api/forum/forum/:forum_id", onAPIUpdateForumForum(env))
	}

	//adminGrp := engine.Group("/")
	//{
	//	// Admin access
	//	adminRoute := adminGrp.Use(authMiddleware(env, domain.PAdmin))
	//	adminRoute.POST("/api/servers", onAPIPostServer(env))
	//	adminRoute.POST("/api/servers/:server_id", onAPIPostServerUpdate(env))
	//	adminRoute.DELETE("/api/servers/:server_id", onAPIPostServerDelete(env))
	//	adminRoute.POST("/api/servers_admin", onAPIGetServersAdmin(env))
	//}

	return engine
}
