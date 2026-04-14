package config

import (
	"context"
	"strings"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/anticheat"
	"github.com/leighmacdonald/gbans/internal/asset"
	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/chat"
	configv1 "github.com/leighmacdonald/gbans/internal/config/v1"
	"github.com/leighmacdonald/gbans/internal/config/v1/configv1connect"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/log"
	"github.com/leighmacdonald/gbans/internal/network"
	"github.com/leighmacdonald/gbans/internal/network/ip2location"
	"github.com/leighmacdonald/gbans/internal/network/scp"
	"github.com/leighmacdonald/gbans/internal/patreon"
	"github.com/leighmacdonald/gbans/internal/ptr"
	"github.com/leighmacdonald/gbans/internal/servers"
	"github.com/leighmacdonald/gbans/internal/sourcemod"
	"google.golang.org/protobuf/types/known/emptypb"
)

type RPC struct {
	configv1connect.UnimplementedConfigServiceHandler
	*Configuration
	version string
}

func NewRPC(conf *Configuration, version string) *RPC {
	return &RPC{
		Configuration: conf,
		version:       version,
	}
}

func (r *RPC) Info(context.Context, *emptypb.Empty) (*configv1.InfoResponse, error) {
	conf := r.Config()

	resp := configv1.InfoResponse{
		SiteName:         &conf.General.SiteName,
		AssetUrl:         &conf.General.AssetURL,
		Favicon:          ptr.To(conf.General.FaviconURL()),
		LinkId:           &conf.Discord.LinkID,
		AppVersion:       &r.version,
		SentryDsnWeb:     &conf.General.SentryDSNWeb,
		SiteDescription:  &conf.General.SiteDescription,
		PatreonClientId:  &conf.Patreon.ClientID,
		DiscordClientId:  &conf.Discord.AppID,
		DiscordEnabled:   &conf.Discord.Enabled,
		PatreonEnabled:   &conf.Patreon.Enabled,
		DefaultRoute:     &conf.General.DefaultRoute,
		NewsEnabled:      &conf.General.NewsEnabled,
		ForumsEnabled:    &conf.General.ForumsEnabled,
		ContestsEnabled:  &conf.General.ContestsEnabled,
		WikiEnabled:      &conf.General.WikiEnabled,
		StatsEnabled:     &conf.General.StatsEnabled,
		ServersEnabled:   &conf.General.ServersEnabled,
		ReportsEnabled:   &conf.General.ReportsEnabled,
		ChatlogsEnabled:  &conf.General.ChatlogsEnabled,
		DemosEnabled:     &conf.General.DemosEnabled,
		SpeedrunsEnabled: &conf.General.SpeedrunsEnabled,
		MgeEnabled:       &conf.General.MGEEnabled,
	}
	return &resp, nil
}

func toRunMode(mode RunMode) configv1.RunMode {
	switch mode {
	case DebugMode:
		return configv1.RunMode_RUN_MODE_DEBUG
	case TestMode:
		return configv1.RunMode_RUN_MODE_TEST
	case ReleaseMode:
		fallthrough
	default:
		return configv1.RunMode_RUN_MODE_RELEASE_UNSPECIFIED
	}
}

func toServeMode(mode FileServeMode) configv1.FileServeMode {
	switch mode {
	case LocalMode:
		fallthrough
	default:
		return configv1.FileServeMode_FILE_SERVE_MODE_LOCAL_UNSPECIFIED
	}
}

func toDemoStrategy(strategy servers.DemoStrategy) configv1.DemoStrategy {
	switch strategy {
	case servers.DemoStrategyCount:
		return configv1.DemoStrategy_DEMO_STRATEGY_COUNT
	case servers.DemoStrategyPctFree:
		fallthrough
	default:
		return configv1.DemoStrategy_DEMO_STRATEGY_PCTFREE_UNSPECIFIED
	}
}

func toLevel(level log.Level) configv1.Level {
	switch level {
	case log.Debug:
		return configv1.Level_LEVEL_DEBUG
	case log.Info:
		return configv1.Level_LEVEL_INFO
	case log.Warn:
		return configv1.Level_LEVEL_WARNING
	case log.Error:
		fallthrough
	default:
		return configv1.Level_LEVEL_ERROR_UNSPECIFIED
	}
}

func (r *RPC) Get(context.Context, *emptypb.Empty) (*configv1.GetResponse, error) {
	c := r.Config()

	return &configv1.GetResponse{
		Config: &configv1.Config{
			General: &configv1.General{
				SiteName:         &c.General.SiteName,
				SiteDescription:  &c.General.SiteDescription,
				Mode:             ptr.To(toRunMode(c.General.Mode)),
				FileServeMode:    ptr.To(toServeMode(c.General.FileServeMode)),
				SrcdsLogAddr:     &c.General.SrcdsLogAddr,
				AssetUrl:         &c.General.AssetURL,
				Favicon:          &c.General.Favicon,
				WikiEnabled:      &c.General.WikiEnabled,
				DefaultRoute:     &c.General.DefaultRoute,
				NewsEnabled:      &c.General.NewsEnabled,
				ForumsEnabled:    &c.General.ForumsEnabled,
				ContestsEnabled:  &c.General.ContestsEnabled,
				StatsEnabled:     &c.General.StatsEnabled,
				ServersEnabled:   &c.General.ServersEnabled,
				ReportsEnabled:   &c.General.ReportsEnabled,
				ChatlogsEnabled:  &c.General.ChatlogsEnabled,
				DemosEnabled:     &c.General.DemosEnabled,
				SpeedrunsEnabled: &c.General.SpeedrunsEnabled,
				SentryDsn:        &c.General.SentryDSN,
				SentryDsnWeb:     &c.General.SentryDSNWeb,
			},
			Discord: &configv1.Discord{
				Token:                   &c.Discord.Token,
				Enabled:                 &c.Discord.Enabled,
				BotEnabled:              &c.Discord.BotEnabled,
				IntegrationsEnabled:     &c.Discord.IntegrationsEnabled,
				AppId:                   &c.Discord.AppID,
				AppSecret:               &c.Discord.AppSecret,
				LinkId:                  &c.Discord.LinkID,
				GuildId:                 &c.Discord.GuildID,
				PublicLogChannelEnable:  &c.Discord.PublicLogChannelEnable,
				LogChannelId:            &c.Discord.LogChannelID,
				PublicMatchLogChannelId: &c.Discord.PublicMatchLogChannelID,
				VoteLogChannelId:        &c.Discord.VoteLogChannelID,
				AppealLogChannelId:      &c.Discord.AppealLogChannelID,
				BanLogChannelId:         &c.Discord.BanLogChannelID,
				ForumLogChannelId:       &c.Discord.ForumLogChannelID,
				KickLogChannelId:        &c.Discord.KickLogChannelID,
				ModPingRoleId:           &c.Discord.ModPingRoleID,
				AnticheatChannelId:      &c.Discord.AnticheatChannelID,
				SeedChannelId:           &c.Discord.SeedChannelID,
				WordFilterLogChannelId:  &c.Discord.WordFilterLogChannelID,
				ChatLogChannelId:        &c.Discord.ChatLogChannelID,
			},
			Patreon: &configv1.Patreon{
				Enabled:             &c.Patreon.Enabled,
				IntegrationsEnabled: &c.Patreon.IntegrationsEnabled,
				ClientId:            &c.Patreon.ClientID,
				ClientSecret:        &c.Patreon.ClientSecret,
				CreatorAccessToken:  &c.Patreon.CreatorAccessToken,
				CreatorRefreshToken: &c.Patreon.CreatorRefreshToken,
			},
			Debug: &configv1.Debug{
				SkipOpenIdValidation: &c.Debug.SkipOpenIDValidation,
				AddRconLogAddress:    &c.Debug.AddRCONLogAddress,
			},
			Demo: &configv1.Demo{
				CleanupEnabled: &c.Demo.DemoCleanupEnabled,
				Strategy:       ptr.To(toDemoStrategy(c.Demo.DemoCleanupStrategy)),
				CleanupMinPct:  &c.Demo.DemoCleanupMinPct,
				CleanupMount:   &c.Demo.DemoCleanupMount,
				CountLimit:     ptr.To(int64(c.Demo.DemoCountLimit)),
				ParserUrl:      &c.Demo.DemoParserURL,
			},
			Filters: &configv1.Filters{
				Enabled:        &c.Filters.Enabled,
				WarningTimeout: ptr.To(int32(c.Filters.WarningTimeout)),
				WarningLimit:   ptr.To(int32(c.Filters.WarningLimit)),
				Dry:            &c.Filters.Dry,
				PingDiscord:    &c.Filters.PingDiscord,
				MaxWeight:      ptr.To(int32(c.Filters.MaxWeight)),
				CheckTimeout:   ptr.To(int32(c.Filters.CheckTimeout)),
				MatchTimeout:   ptr.To(int32(c.Filters.MatchTimeout)),
			},
			Log: &configv1.Log{
				Level:           ptr.To(toLevel(c.Log.Level)),
				File:            &c.Log.File,
				HttpEnabled:     &c.Log.HTTPEnabled,
				HttpOtelEnabled: &c.Log.HTTPOtelEnabled,
				HttpLevel:       ptr.To(toLevel(c.Log.HTTPLevel)),
			},
			GeoLocation: &configv1.GeoLocation{
				Enabled:   &c.GeoLocation.Enabled,
				CachePath: &c.GeoLocation.CachePath,
				Token:     &c.GeoLocation.Token,
			},
			Ssh: &configv1.SSH{
				Enabled:         &c.SSH.Enabled,
				Username:        &c.SSH.Username,
				Port:            ptr.To(int32(c.SSH.Port)),
				PrivateKeyPath:  &c.SSH.PrivateKeyPath,
				HostKeyStrategy: ptr.To(configv1.HostKeyStrategy(c.SSH.HostKeyStrategy)),
				Password:        &c.SSH.Password,
				UpdateInterval:  ptr.To(int32(c.SSH.UpdateInterval)),
				Timeout:         ptr.To(int32(c.SSH.Timeout)),
				DemoPathFmt:     &c.SSH.DemoPathFmt,
				StacPath:        &c.SSH.StacPathFmt,
			},
			Network: &configv1.Network{
				SdrEnabled:    &c.Network.SDREnabled,
				SdrDnsEnabled: &c.Network.SDRDNSEnabled,
				CfKey:         &c.Network.CFKey,
				CfEmail:       &c.Network.CFEmail,
				CfZoneId:      &c.Network.CFZoneID,
			},
			LocalStore: &configv1.LocalStore{
				PathRoot: &c.LocalStore.PathRoot,
			},
			Exports: &configv1.Exports{
				BdEnabled:      &c.Exports.BDEnabled,
				ValveEnabled:   &c.Exports.ValveEnabled,
				AuthorizedKeys: strings.Split(c.Exports.AuthorizedKeys, ","),
			},
			Anticheat: &configv1.Anticheat{
				Enabled:               &c.Anticheat.Enabled,
				Action:                ptr.To(toAction(c.Anticheat.Action)),
				Duration:              ptr.To(int32(c.Anticheat.Duration)),
				MaxAimSnaps:           ptr.To(int32(c.Anticheat.MaxAimSnap)),
				MaxPsilent:            ptr.To(int32(c.Anticheat.MaxPsilent)),
				MaxBhop:               ptr.To(int32(c.Anticheat.MaxBhop)),
				MaxFakeAng:            ptr.To(int32(c.Anticheat.MaxFakeAng)),
				MaxCmdNum:             ptr.To(int32(c.Anticheat.MaxCmdNum)),
				MaxTooManyConnections: ptr.To(int32(c.Anticheat.MaxTooManyConnections)),
				MaxOobVar:             ptr.To(int32(c.Anticheat.MaxOOBVar)),
				MaxInvalidUserCmd:     ptr.To(int32(c.Anticheat.MaxInvalidUserCmd)),
			},
		},
	}, nil
}

func toAction(action anticheat.Action) configv1.Action {
	switch action {
	case anticheat.ActionGag:
		return configv1.Action_ACTION_GAG
	case anticheat.ActionBan:
		return configv1.Action_ACTION_BAN
	case anticheat.ActionKick:
		fallthrough
	default:
		return configv1.Action_ACTION_KICK_UNSPECIFIED

	}
}

func (r *RPC) Update(ctx context.Context, request *configv1.UpdateRequest) (*configv1.UpdateResponse, error) {
	conf := Config{
		General: General{
			SiteName:         ptr.From(request.Config.General.SiteName),
			SiteDescription:  ptr.From(request.Config.General.SiteDescription),
			Mode:             RunMode(request.Config.General.Mode.String()),
			FileServeMode:    FileServeMode(request.Config.General.FileServeMode.String()),
			SrcdsLogAddr:     ptr.From(request.Config.General.SrcdsLogAddr),
			AssetURL:         ptr.From(request.Config.General.AssetUrl),
			Favicon:          ptr.From(request.Config.General.Favicon),
			DefaultRoute:     ptr.From(request.Config.General.DefaultRoute),
			NewsEnabled:      ptr.From(request.Config.General.NewsEnabled),
			ForumsEnabled:    ptr.From(request.Config.General.ForumsEnabled),
			ContestsEnabled:  ptr.From(request.Config.General.ContestsEnabled),
			WikiEnabled:      ptr.From(request.Config.General.WikiEnabled),
			StatsEnabled:     ptr.From(request.Config.General.StatsEnabled),
			ServersEnabled:   ptr.From(request.Config.General.ServersEnabled),
			ReportsEnabled:   ptr.From(request.Config.General.ReportsEnabled),
			ChatlogsEnabled:  ptr.From(request.Config.General.ChatlogsEnabled),
			DemosEnabled:     ptr.From(request.Config.General.DemosEnabled),
			SpeedrunsEnabled: ptr.From(request.Config.General.SpeedrunsEnabled),
			SentryDSN:        ptr.From(request.Config.General.SentryDsn),
			SentryDSNWeb:     ptr.From(request.Config.General.SentryDsnWeb),
		},
		Debug: Debug{
			SkipOpenIDValidation: ptr.From(request.Config.Debug.SkipOpenIdValidation),
			AddRCONLogAddress:    ptr.From(request.Config.Debug.AddRconLogAddress),
		},
		Demo: servers.DemoConfig{
			DemoCleanupEnabled:  ptr.From(request.Config.Demo.CleanupEnabled),
			DemoCleanupStrategy: servers.DemoStrategy(request.Config.Demo.Strategy.String()),
			DemoCleanupMinPct:   ptr.From(request.Config.Demo.CleanupMinPct),
			DemoCleanupMount:    ptr.From(request.Config.Demo.CleanupMount),
			DemoCountLimit:      uint64(ptr.From(request.Config.Demo.CountLimit)),
			DemoParserURL:       ptr.From(request.Config.Demo.ParserUrl),
		},
		Filters: chat.Config{
			Enabled:        ptr.From(request.Config.Filters.Enabled),
			WarningTimeout: int(ptr.From(request.Config.Filters.WarningTimeout)),
			WarningLimit:   int(ptr.From(request.Config.Filters.WarningLimit)),
			Dry:            ptr.From(request.Config.Filters.Dry),
			PingDiscord:    ptr.From(request.Config.Filters.PingDiscord),
			MaxWeight:      ptr.From(request.Config.Filters.MaxWeight),
			CheckTimeout:   int(ptr.From(request.Config.Filters.CheckTimeout)),
			MatchTimeout:   int(ptr.From(request.Config.Filters.MatchTimeout)),
		},
		Discord: discord.Config{
			Enabled:                 ptr.From(request.Config.Discord.Enabled),
			BotEnabled:              ptr.From(request.Config.Discord.BotEnabled),
			IntegrationsEnabled:     ptr.From(request.Config.Discord.IntegrationsEnabled),
			AppID:                   ptr.From(request.Config.Discord.AppId),
			AppSecret:               ptr.From(request.Config.Discord.AppSecret),
			LinkID:                  ptr.From(request.Config.Discord.LinkId),
			Token:                   ptr.From(request.Config.Discord.Token),
			GuildID:                 ptr.From(request.Config.Discord.GuildId),
			PublicLogChannelEnable:  ptr.From(request.Config.Discord.PublicLogChannelEnable),
			LogChannelID:            ptr.From(request.Config.Discord.LogChannelId),
			PublicLogChannelID:      ptr.From(request.Config.Discord.PublicMatchLogChannelId),
			PublicMatchLogChannelID: ptr.From(request.Config.Discord.PublicMatchLogChannelId),
			VoteLogChannelID:        ptr.From(request.Config.Discord.VoteLogChannelId),
			AppealLogChannelID:      ptr.From(request.Config.Discord.AppealLogChannelId),
			BanLogChannelID:         ptr.From(request.Config.Discord.BanLogChannelId),
			ForumLogChannelID:       ptr.From(request.Config.Discord.ForumLogChannelId),
			KickLogChannelID:        ptr.From(request.Config.Discord.KickLogChannelId),
			ModPingRoleID:           ptr.From(request.Config.Discord.ModPingRoleId),
			AnticheatChannelID:      ptr.From(request.Config.Discord.AnticheatChannelId),
			SeedChannelID:           ptr.From(request.Config.Discord.SeedChannelId),
			WordFilterLogChannelID:  ptr.From(request.Config.Discord.WordFilterLogChannelId),
			ChatLogChannelID:        ptr.From(request.Config.Discord.ChatLogChannelId),
		},
		Clientprefs: sourcemod.Config{
			CenterProjectiles: ptr.From(request.Config.ClientPrefs.CenterProjectiles),
		},
		Log: log.Config{
			Level:           log.Level(request.Config.Log.Level.String()),
			File:            *request.Config.Log.File,
			HTTPEnabled:     ptr.From(request.Config.Log.HttpEnabled),
			HTTPOtelEnabled: ptr.From(request.Config.Log.HttpOtelEnabled),
			HTTPLevel:       log.Level(request.Config.Log.HttpLevel.String()),
		},
		GeoLocation: ip2location.Config{
			Enabled:   ptr.From(request.Config.GeoLocation.Enabled),
			CachePath: ptr.From(request.Config.GeoLocation.CachePath),
			Token:     ptr.From(request.Config.GeoLocation.Token),
		},
		Patreon: patreon.Config{
			Enabled:             ptr.From(request.Config.Patreon.Enabled),
			IntegrationsEnabled: ptr.From(request.Config.Patreon.IntegrationsEnabled),
			ClientID:            ptr.From(request.Config.Patreon.ClientId),
			ClientSecret:        ptr.From(request.Config.Patreon.ClientSecret),
			CreatorAccessToken:  ptr.From(request.Config.Patreon.CreatorAccessToken),
			CreatorRefreshToken: ptr.From(request.Config.Patreon.CreatorRefreshToken),
		},
		SSH: scp.Config{
			Enabled:         ptr.From(request.Config.Ssh.Enabled),
			Username:        ptr.From(request.Config.Ssh.Username),
			Port:            int(ptr.From(request.Config.Ssh.Port)),
			PrivateKeyPath:  ptr.From(request.Config.Ssh.PrivateKeyPath),
			HostKeyStrategy: scp.HostKeyStrategy(ptr.From(request.Config.Ssh.HostKeyStrategy)),
			Password:        ptr.From(request.Config.Ssh.Password),
			UpdateInterval:  int(ptr.From(request.Config.Ssh.UpdateInterval)),
			Timeout:         int(ptr.From(request.Config.Ssh.Timeout)),
			DemoPathFmt:     ptr.From(request.Config.Ssh.DemoPathFmt),
			StacPathFmt:     ptr.From(request.Config.Ssh.StacPath),
		},
		Network: network.Config{
			SDREnabled:    ptr.From(request.Config.Network.SdrEnabled),
			SDRDNSEnabled: ptr.From(request.Config.Network.SdrDnsEnabled),
			CFKey:         ptr.From(request.Config.Network.CfKey),
			CFEmail:       ptr.From(request.Config.Network.CfEmail),
			CFZoneID:      ptr.From(request.Config.Network.CfZoneId),
		},
		LocalStore: asset.Config{
			PathRoot: ptr.From(request.Config.LocalStore.PathRoot),
		},
		Exports: ban.Config{
			BDEnabled:      ptr.From(request.Config.Exports.BdEnabled),
			ValveEnabled:   ptr.From(request.Config.Exports.ValveEnabled),
			AuthorizedKeys: strings.Join(request.Config.Exports.AuthorizedKeys, ","),
		},
		Anticheat: anticheat.Config{
			Enabled:               ptr.From(request.Config.Anticheat.Enabled),
			Action:                anticheat.Action(request.Config.Anticheat.Action.String()),
			Duration:              int(ptr.From(request.Config.Anticheat.Duration)),
			MaxAimSnap:            int(ptr.From(request.Config.Anticheat.MaxAimSnaps)),
			MaxPsilent:            int(ptr.From(request.Config.Anticheat.MaxPsilent)),
			MaxBhop:               int(ptr.From(request.Config.Anticheat.MaxBhop)),
			MaxFakeAng:            int(ptr.From(request.Config.Anticheat.MaxFakeAng)),
			MaxCmdNum:             int(ptr.From(request.Config.Anticheat.MaxCmdNum)),
			MaxTooManyConnections: int(ptr.From(request.Config.Anticheat.MaxTooManyConnections)),
			MaxCheatCvar:          int(ptr.From(request.Config.Anticheat.MaxCheatCvar)),
			MaxOOBVar:             int(ptr.From(request.Config.Anticheat.MaxOobVar)),
			MaxInvalidUserCmd:     int(ptr.From(request.Config.Anticheat.MaxInvalidUserCmd)),
		},
	}
	if errWrite := r.Configuration.Write(ctx, conf); errWrite != nil {
		return nil, connect.NewError(connect.CodeUnknown, errWrite)
	}

	return nil, nil
}
