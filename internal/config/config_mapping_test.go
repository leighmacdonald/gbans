//nolint:testpackage // Must be internal to cover the unexported mapping helpers.
package config

import (
	"testing"

	"github.com/leighmacdonald/gbans/internal/anticheat"
	"github.com/leighmacdonald/gbans/internal/asset"
	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/chat"
	configv1 "github.com/leighmacdonald/gbans/internal/config/v1"
	"github.com/leighmacdonald/gbans/internal/demo"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/log"
	"github.com/leighmacdonald/gbans/internal/network"
	"github.com/leighmacdonald/gbans/internal/network/ip2location"
	"github.com/leighmacdonald/gbans/internal/network/scp"
	"github.com/leighmacdonald/gbans/internal/patreon"
	"github.com/leighmacdonald/gbans/internal/sourcemod"
	"github.com/stretchr/testify/require"
)

func testDomainConfig() Config {
	return Config{
		General: &General{
			SiteName: "name", SiteDescription: "desc", Mode: DebugMode, FileServeMode: LocalMode,
			SrcdsLogAddr: "127.0.0.1:27000", AssetURL: "https://example.com/assets/", Favicon: "icon.png",
			DefaultRoute: "/servers", NewsEnabled: true, ForumsEnabled: true, ContestsEnabled: true,
			WikiEnabled: true, StatsEnabled: true, ServersEnabled: true, ReportsEnabled: true,
			ChatlogsEnabled: true, DemosEnabled: true, PlayerqueueEnabled: true, SpeedrunsEnabled: true,
			SentryDSN: "sentry-dsn", SentryDSNWeb: "sentry-dsn-web", MGEEnabled: true,
		},
		Debug: &Debug{SkipOpenIDValidation: true, AddRCONLogAddress: "127.0.0.1:27015"},
		Demo: &demo.Config{
			DemoCleanupEnabled: true, DemoCleanupStrategy: demo.DemoStrategyCount, DemoCleanupMinPct: 25,
			DemoCleanupMount: "/mnt/demos", DemoCountLimit: 1000, DemoParserURL: "https://parser.example.com",
		},
		Filters: &chat.Config{
			Enabled: true, WarningTimeout: 60, WarningLimit: 3, Dry: true, PingDiscord: true,
			MaxWeight: 10, CheckTimeout: 5, MatchTimeout: 30,
		},
		Discord: &discord.Config{
			Enabled: true, BotEnabled: true, IntegrationsEnabled: true, AppID: "app-id", AppSecret: "app-secret",
			LinkID: "link-id", Token: "token", GuildID: "guild-id", PublicLogChannelEnable: true,
			LogChannelID: "log-channel", PublicLogChannelID: "public-log-channel",
			PublicMatchLogChannelID: "match-log-channel", VoteLogChannelID: "vote-log-channel",
			AppealLogChannelID: "appeal-log-channel", BanLogChannelID: "ban-log-channel",
			ForumLogChannelID: "forum-log-channel", KickLogChannelID: "kick-log-channel",
			ModPingRoleID: "ping-role", AnticheatChannelID: "anticheat-channel", SeedChannelID: "seed-channel",
			WordFilterLogChannelID: "filter-log-channel", ChatLogChannelID: "chat-log-channel",
		},
		Clientprefs: &sourcemod.Config{CenterProjectiles: true},
		Log:         &log.Config{Level: log.Info, File: "/var/log/gbans.log", HTTPEnabled: true, HTTPOtelEnabled: true, HTTPLevel: log.Debug},
		GeoLocation: &ip2location.Config{Enabled: true, CachePath: "/tmp/geo", Token: "geo-token"},
		Patreon: &patreon.Config{
			Enabled: true, IntegrationsEnabled: true, ClientID: "patreon-id", ClientSecret: "patreon-secret",
			CreatorAccessToken: "access-token", CreatorRefreshToken: "refresh-token",
		},
		SSH: &scp.Config{
			Enabled: true, Username: "user", Port: 2222, PrivateKeyPath: "/home/user/.ssh/id",
			HostKeyStrategy: scp.KeyAutoAcceptFirst, Password: "ssh-password", UpdateInterval: 60,
			Timeout: 10, DemoPathFmt: "/demos/%s", StacPathFmt: "/stac/%s",
		},
		Network:    &network.Config{SDREnabled: true},
		LocalStore: &asset.Config{PathRoot: "/var/lib/gbans"},
		Exports:    &ban.Config{BDEnabled: true, ValveEnabled: true, AuthorizedKeys: "key-1,key-2"},
		Anticheat: &anticheat.Config{
			Enabled: true, Action: anticheat.ActionBan, Duration: 86400, MaxAimSnap: 5, MaxPsilent: 5,
			MaxBhop: 40, MaxFakeAng: 5, MaxCmdNum: 100, MaxTooManyConnections: 10, MaxCheatCvar: 5,
			MaxOOBVar: 5, MaxInvalidUserCmd: 5,
		},
	}
}

func TestConfigMappingRoundTrip(t *testing.T) {
	original := testDomainConfig()

	// Domain -> api -> domain must preserve every field so that saving the
	// settings form can never wipe or corrupt values.
	mapped := configFromUpdate(toConfig(original), Config{
		General:     &General{},
		Clientprefs: &sourcemod.Config{CenterProjectiles: true},
	})

	require.Equal(t, original.General.SiteName, mapped.General.SiteName)
	require.Equal(t, original.General.SiteDescription, mapped.General.SiteDescription)
	require.Equal(t, original.General.Mode, mapped.General.Mode)
	require.Equal(t, original.General.FileServeMode, mapped.General.FileServeMode)
	require.Equal(t, original.General.SrcdsLogAddr, mapped.General.SrcdsLogAddr)
	require.Equal(t, original.General.AssetURL, mapped.General.AssetURL)
	require.Equal(t, original.General.Favicon, mapped.General.Favicon)
	require.Equal(t, original.General.DefaultRoute, mapped.General.DefaultRoute)
	require.Equal(t, original.General.NewsEnabled, mapped.General.NewsEnabled)
	require.Equal(t, original.General.ForumsEnabled, mapped.General.ForumsEnabled)
	require.Equal(t, original.General.ContestsEnabled, mapped.General.ContestsEnabled)
	require.Equal(t, original.General.WikiEnabled, mapped.General.WikiEnabled)
	require.Equal(t, original.General.StatsEnabled, mapped.General.StatsEnabled)
	require.Equal(t, original.General.ServersEnabled, mapped.General.ServersEnabled)
	require.Equal(t, original.General.ReportsEnabled, mapped.General.ReportsEnabled)
	require.Equal(t, original.General.ChatlogsEnabled, mapped.General.ChatlogsEnabled)
	require.Equal(t, original.General.DemosEnabled, mapped.General.DemosEnabled)
	require.Equal(t, original.General.SpeedrunsEnabled, mapped.General.SpeedrunsEnabled)
	require.Equal(t, original.General.SentryDSN, mapped.General.SentryDSN)
	require.Equal(t, original.General.SentryDSNWeb, mapped.General.SentryDSNWeb)
	require.Equal(t, original.General.MGEEnabled, mapped.General.MGEEnabled)

	require.Equal(t, original.Debug, mapped.Debug)
	require.Equal(t, original.Demo, mapped.Demo)
	require.Equal(t, original.Filters, mapped.Filters)
	require.Equal(t, original.Log, mapped.Log)
	require.Equal(t, original.GeoLocation, mapped.GeoLocation)
	require.Equal(t, original.Patreon, mapped.Patreon)
	require.Equal(t, original.SSH, mapped.SSH)
	require.Equal(t, original.Network, mapped.Network)
	require.Equal(t, original.LocalStore, mapped.LocalStore)
	require.Equal(t, original.Exports, mapped.Exports)
	require.Equal(t, original.Anticheat, mapped.Anticheat)

	// Regression: the secret was mapped from the app id, destroying it on save.
	require.Equal(t, "app-secret", mapped.Discord.AppSecret)
	// Regression: the public log channel id was mapped from the match log channel.
	require.Equal(t, "public-log-channel", mapped.Discord.PublicLogChannelID)
	require.Equal(t, original.Discord, mapped.Discord)

	// Fields not managed through the api must be carried over, not zeroed.
	current := Config{General: &General{PlayerqueueEnabled: true}, Clientprefs: &sourcemod.Config{CenterProjectiles: true}}
	preserved := configFromUpdate(&configv1.Config{}, current)
	require.True(t, preserved.General.PlayerqueueEnabled)
	require.True(t, preserved.Clientprefs.CenterProjectiles)
}
