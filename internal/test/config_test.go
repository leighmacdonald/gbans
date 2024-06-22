package test_test

import (
	"net/http"
	"testing"

	"github.com/leighmacdonald/gbans/internal/domain"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {
	router := testRouter()
	owner := loginUser(getOwner())

	var config domain.Config
	testEndpointWithReceiver(t, router, http.MethodGet, "/api/config", nil, http.StatusOK, owner, &config)
	config.StaticConfig = configUC.Config().StaticConfig
	require.EqualValues(t, configUC.Config(), config)

	config.General.SiteName += "x"
	config.General.Mode = domain.DebugMode
	config.General.FileServeMode = "local"
	config.General.SrcdsLogAddr += "x"
	config.General.AssetURL += "x"
	config.General.DefaultRoute = "/"
	config.General.NewsEnabled = !config.General.NewsEnabled
	config.General.ForumsEnabled = !config.General.ForumsEnabled
	config.General.ContestsEnabled = !config.General.ContestsEnabled
	config.General.WikiEnabled = !config.General.WikiEnabled
	config.General.StatsEnabled = !config.General.StatsEnabled
	config.General.ServersEnabled = !config.General.ServersEnabled
	config.General.ReportsEnabled = !config.General.ReportsEnabled
	config.General.ChatlogsEnabled = !config.General.ChatlogsEnabled
	config.General.DemosEnabled = !config.General.DemosEnabled

	config.Debug.SkipOpenIDValidation = !config.Debug.SkipOpenIDValidation
	config.Debug.AddRCONLogAddress = "1.2.3.4:27115"

	config.Log.Level = log.Warn
	config.Log.File += "x"
	config.Log.HTTPLevel = log.Warn
	config.Log.HTTPEnabled = !config.Log.HTTPEnabled
	config.Log.HTTPOtelEnabled = !config.Log.HTTPOtelEnabled

	config.Patreon.ClientID += "x"
	config.Patreon.ClientSecret += "x"
	config.Patreon.IntegrationsEnabled = !config.Patreon.IntegrationsEnabled
	config.Patreon.CreatorAccessToken += "x"
	config.Patreon.CreatorRefreshToken += "x"
	config.Patreon.CreatorAccessToken += "x"

	config.Discord.Enabled = !config.Discord.Enabled
	config.Discord.IntegrationsEnabled = !config.Discord.IntegrationsEnabled
	config.Discord.BotEnabled = !config.Discord.BotEnabled
	config.Discord.AppID += "x"
	config.Discord.AppSecret += "x"
	config.Discord.LinkID += "x"
	config.Discord.Token += "x"
	config.Discord.GuildID += "x"
	config.Discord.LogChannelID += "x"
	config.Discord.PublicLogChannelEnable = !config.Discord.PublicLogChannelEnable
	config.Discord.PublicLogChannelID += "x"
	config.Discord.PublicMatchLogChannelID += "x"
	config.Discord.VoteLogChannelID += "x"
	config.Discord.AppealLogChannelID += "x"
	config.Discord.BanLogChannelID += "x"
	config.Discord.ForumLogChannelID += "x"
	config.Discord.WordFilterLogChannelID += "x"
	config.Discord.KickLogChannelID += "x"
	config.Discord.ModPingRoleID += "x"
	config.Discord.UnregisterOnStart = !config.Discord.UnregisterOnStart

	config.Sentry.SentryDSN += "x"
	config.Sentry.SentryDSN += "x"
	config.Sentry.SentryTrace = !config.Sentry.SentryTrace
	config.Sentry.SentrySampleRate *= 2

	config.Filters.Enabled = !config.Filters.Enabled
	config.Filters.WarningTimeout *= 2
	config.Filters.WarningLimit *= 2
	config.Filters.Dry = !config.Filters.Dry
	config.Filters.PingDiscord = !config.Filters.PingDiscord
	config.Filters.MaxWeight *= 2
	config.Filters.CheckTimeout *= 2
	config.Filters.MatchTimeout *= 2

	config.GeoLocation.Enabled = !config.GeoLocation.Enabled
	config.GeoLocation.CachePath += "x"
	config.GeoLocation.Token += "x"

	config.Demo.DemoCleanupEnabled = !config.Demo.DemoCleanupEnabled
	config.Demo.DemoCleanupStrategy = "count"
	config.Demo.DemoCleanupMinPct *= 2
	config.Demo.DemoCleanupMount += "/x"
	config.Demo.DemoCountLimit *= 2

	config.Clientprefs.CenterProjectiles = !config.Clientprefs.CenterProjectiles

	config.LocalStore.PathRoot += "/x"

	config.Exports.AuthorizedKeys = append(config.Exports.AuthorizedKeys, "test-key")
	config.Exports.BDEnabled = !config.Exports.BDEnabled
	config.Exports.ValveEnabled = !config.Exports.ValveEnabled

	config.SSH.Enabled = !config.SSH.Enabled
	config.SSH.Username += "x"
	config.SSH.Port += 2
	config.SSH.PrivateKeyPath += "/x"
	config.SSH.Password += "x"
	config.SSH.UpdateInterval *= 2
	config.SSH.Timeout *= 2
	config.SSH.DemoPathFmt += "x"

	var updated domain.Config
	testEndpointWithReceiver(t, router, http.MethodPut, "/api/config", config, http.StatusOK, owner, &updated)
	updated.StaticConfig = configUC.Config().StaticConfig
	require.EqualValues(t, config, updated)
}

func TestConfigPermissions(t *testing.T) {
	testPermissions(t, testRouter(), []permTestValues{
		{
			path:   "/api/config",
			method: http.MethodGet,
			code:   http.StatusForbidden,
			levels: admin,
		},
		{
			path:   "/api/config",
			method: http.MethodPut,
			code:   http.StatusForbidden,
			levels: admin,
		},
	})
}
