package tests_test

import (
	"net/http"
	"testing"

	"github.com/leighmacdonald/gbans/internal/config"
	"github.com/leighmacdonald/gbans/pkg/log"
	"github.com/stretchr/testify/require"
)

func TestConfig(t *testing.T) {
	router := testRouter()
	owner := loginUser(getOwner())

	var conf config.Config
	testEndpointWithReceiver(t, router, http.MethodGet, "/api/config", nil, http.StatusOK, &authTokens{user: owner}, &conf)
	conf.StaticConfig = configUC.Config().StaticConfig
	require.Equal(t, configUC.Config(), conf)

	conf.General.SiteName += "x"
	conf.General.Mode = config.TestMode
	conf.General.FileServeMode = "local"
	conf.General.SrcdsLogAddr += "x"
	conf.General.AssetURL += "x"
	conf.General.DefaultRoute = "/"
	conf.General.NewsEnabled = !conf.General.NewsEnabled
	conf.General.ForumsEnabled = !conf.General.ForumsEnabled
	conf.General.ContestsEnabled = !conf.General.ContestsEnabled
	conf.General.WikiEnabled = !conf.General.WikiEnabled
	conf.General.StatsEnabled = !conf.General.StatsEnabled
	conf.General.ServersEnabled = !conf.General.ServersEnabled
	conf.General.ReportsEnabled = !conf.General.ReportsEnabled
	conf.General.ChatlogsEnabled = !conf.General.ChatlogsEnabled
	conf.General.DemosEnabled = !conf.General.DemosEnabled

	conf.Debug.SkipOpenIDValidation = !conf.Debug.SkipOpenIDValidation
	conf.Debug.AddRCONLogAddress = "1.2.3.4:27715"

	conf.Log.Level = log.Warn
	conf.Log.File += "x"
	conf.Log.HTTPLevel = log.Warn
	conf.Log.HTTPEnabled = !conf.Log.HTTPEnabled
	conf.Log.HTTPOtelEnabled = !conf.Log.HTTPOtelEnabled

	conf.Patreon.ClientID += "x"
	conf.Patreon.ClientSecret += "x"
	conf.Patreon.IntegrationsEnabled = !conf.Patreon.IntegrationsEnabled
	conf.Patreon.CreatorAccessToken += "x"
	conf.Patreon.CreatorRefreshToken += "x"
	conf.Patreon.CreatorAccessToken += "x"

	conf.Discord.Enabled = !conf.Discord.Enabled
	conf.Discord.IntegrationsEnabled = !conf.Discord.IntegrationsEnabled
	conf.Discord.BotEnabled = !conf.Discord.BotEnabled
	conf.Discord.AppID += "x"
	conf.Discord.AppSecret += "x"
	conf.Discord.LinkID += "x"
	conf.Discord.Token += "x"
	conf.Discord.GuildID += "x"
	conf.Discord.LogChannelID += "x"
	conf.Discord.PublicLogChannelEnable = !conf.Discord.PublicLogChannelEnable
	conf.Discord.PublicLogChannelID += "x"
	conf.Discord.PublicMatchLogChannelID += "x"
	conf.Discord.VoteLogChannelID += "x"
	conf.Discord.AppealLogChannelID += "x"
	conf.Discord.BanLogChannelID += "x"
	conf.Discord.ForumLogChannelID += "x"
	conf.Discord.WordFilterLogChannelID += "x"
	conf.Discord.KickLogChannelID += "x"
	conf.Discord.ModPingRoleID += "x"

	conf.Filters.Enabled = !conf.Filters.Enabled
	conf.Filters.WarningTimeout *= 2
	conf.Filters.WarningLimit *= 2
	conf.Filters.Dry = !conf.Filters.Dry
	conf.Filters.PingDiscord = !conf.Filters.PingDiscord
	conf.Filters.MaxWeight *= 2
	conf.Filters.CheckTimeout *= 2
	conf.Filters.MatchTimeout *= 2

	conf.GeoLocation.Enabled = !conf.GeoLocation.Enabled
	conf.GeoLocation.CachePath += "x"
	conf.GeoLocation.Token += "x"

	conf.Demo.DemoCleanupEnabled = !conf.Demo.DemoCleanupEnabled
	conf.Demo.DemoCleanupStrategy = "count"
	conf.Demo.DemoCleanupMinPct *= 2
	conf.Demo.DemoCleanupMount += "/x"
	conf.Demo.DemoCountLimit *= 2

	conf.Clientprefs.CenterProjectiles = !conf.Clientprefs.CenterProjectiles

	conf.LocalStore.PathRoot += "/x"

	conf.Exports.AuthorizedKeys += ",test-key"
	conf.Exports.BDEnabled = !conf.Exports.BDEnabled
	conf.Exports.ValveEnabled = !conf.Exports.ValveEnabled

	conf.SSH.Enabled = !conf.SSH.Enabled
	conf.SSH.Username += "x"
	conf.SSH.Port += 2
	conf.SSH.PrivateKeyPath += "/x"
	conf.SSH.Password += "x"
	conf.SSH.UpdateInterval *= 2
	conf.SSH.Timeout *= 2
	conf.SSH.DemoPathFmt += "x"

	var updated config.Config
	testEndpointWithReceiver(t, router, http.MethodPut, "/api/config", conf, http.StatusOK, &authTokens{user: owner}, &updated)
	updated.StaticConfig = configUC.Config().StaticConfig
	require.Equal(t, conf, updated)
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
