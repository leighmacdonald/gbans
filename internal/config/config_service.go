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

type Service struct {
	configv1connect.UnimplementedConfigServiceHandler

	config  *Configuration
	version string
}

func NewService(conf *Configuration, version string) *Service {
	return &Service{
		config:  conf,
		version: version,
	}
}

func (r *Service) Info(context.Context, *emptypb.Empty) (*configv1.InfoResponse, error) {
	conf := r.config.Config()

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

func (r *Service) Get(context.Context, *emptypb.Empty) (*configv1.GetResponse, error) {
	return &configv1.GetResponse{Config: toConfig(r.config.Config())}, nil
}

func (r *Service) Update(ctx context.Context, request *configv1.UpdateRequest) (*configv1.UpdateResponse, error) {
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
	if errWrite := r.config.Write(ctx, conf); errWrite != nil {
		return nil, connect.NewError(connect.CodeUnknown, errWrite)
	}

	return &configv1.UpdateResponse{Config: toConfig(conf)}, nil
}

func toConfig(conf Config) *configv1.Config {
	return &configv1.Config{
		General: &configv1.General{
			SiteName:         &conf.General.SiteName,
			SiteDescription:  &conf.General.SiteDescription,
			Mode:             ptr.To(toRunMode(conf.General.Mode)),
			FileServeMode:    ptr.To(toServeMode(conf.General.FileServeMode)),
			SrcdsLogAddr:     &conf.General.SrcdsLogAddr,
			AssetUrl:         &conf.General.AssetURL,
			Favicon:          &conf.General.Favicon,
			WikiEnabled:      &conf.General.WikiEnabled,
			DefaultRoute:     &conf.General.DefaultRoute,
			NewsEnabled:      &conf.General.NewsEnabled,
			ForumsEnabled:    &conf.General.ForumsEnabled,
			ContestsEnabled:  &conf.General.ContestsEnabled,
			StatsEnabled:     &conf.General.StatsEnabled,
			ServersEnabled:   &conf.General.ServersEnabled,
			ReportsEnabled:   &conf.General.ReportsEnabled,
			ChatlogsEnabled:  &conf.General.ChatlogsEnabled,
			DemosEnabled:     &conf.General.DemosEnabled,
			SpeedrunsEnabled: &conf.General.SpeedrunsEnabled,
			SentryDsn:        &conf.General.SentryDSN,
			SentryDsnWeb:     &conf.General.SentryDSNWeb,
		},
		Discord: &configv1.Discord{
			Token:                   &conf.Discord.Token,
			Enabled:                 &conf.Discord.Enabled,
			BotEnabled:              &conf.Discord.BotEnabled,
			IntegrationsEnabled:     &conf.Discord.IntegrationsEnabled,
			AppId:                   &conf.Discord.AppID,
			AppSecret:               &conf.Discord.AppSecret,
			LinkId:                  &conf.Discord.LinkID,
			GuildId:                 &conf.Discord.GuildID,
			PublicLogChannelEnable:  &conf.Discord.PublicLogChannelEnable,
			LogChannelId:            &conf.Discord.LogChannelID,
			PublicMatchLogChannelId: &conf.Discord.PublicMatchLogChannelID,
			VoteLogChannelId:        &conf.Discord.VoteLogChannelID,
			AppealLogChannelId:      &conf.Discord.AppealLogChannelID,
			BanLogChannelId:         &conf.Discord.BanLogChannelID,
			ForumLogChannelId:       &conf.Discord.ForumLogChannelID,
			KickLogChannelId:        &conf.Discord.KickLogChannelID,
			ModPingRoleId:           &conf.Discord.ModPingRoleID,
			AnticheatChannelId:      &conf.Discord.AnticheatChannelID,
			SeedChannelId:           &conf.Discord.SeedChannelID,
			WordFilterLogChannelId:  &conf.Discord.WordFilterLogChannelID,
			ChatLogChannelId:        &conf.Discord.ChatLogChannelID,
		},
		Patreon: &configv1.Patreon{
			Enabled:             &conf.Patreon.Enabled,
			IntegrationsEnabled: &conf.Patreon.IntegrationsEnabled,
			ClientId:            &conf.Patreon.ClientID,
			ClientSecret:        &conf.Patreon.ClientSecret,
			CreatorAccessToken:  &conf.Patreon.CreatorAccessToken,
			CreatorRefreshToken: &conf.Patreon.CreatorRefreshToken,
		},
		Debug: &configv1.Debug{
			SkipOpenIdValidation: &conf.Debug.SkipOpenIDValidation,
			AddRconLogAddress:    &conf.Debug.AddRCONLogAddress,
		},
		Demo: &configv1.Demo{
			CleanupEnabled: &conf.Demo.DemoCleanupEnabled,
			Strategy:       ptr.To(toDemoStrategy(conf.Demo.DemoCleanupStrategy)),
			CleanupMinPct:  &conf.Demo.DemoCleanupMinPct,
			CleanupMount:   &conf.Demo.DemoCleanupMount,
			CountLimit:     ptr.To(int64(conf.Demo.DemoCountLimit)),
			ParserUrl:      &conf.Demo.DemoParserURL,
		},
		Filters: &configv1.Filters{
			Enabled:        &conf.Filters.Enabled,
			WarningTimeout: ptr.To(int32(conf.Filters.WarningTimeout)),
			WarningLimit:   ptr.To(int32(conf.Filters.WarningLimit)),
			Dry:            &conf.Filters.Dry,
			PingDiscord:    &conf.Filters.PingDiscord,
			MaxWeight:      ptr.To(conf.Filters.MaxWeight),
			CheckTimeout:   ptr.To(int32(conf.Filters.CheckTimeout)),
			MatchTimeout:   ptr.To(int32(conf.Filters.MatchTimeout)),
		},
		Log: &configv1.Log{
			Level:           ptr.To(toLevel(conf.Log.Level)),
			File:            &conf.Log.File,
			HttpEnabled:     &conf.Log.HTTPEnabled,
			HttpOtelEnabled: &conf.Log.HTTPOtelEnabled,
			HttpLevel:       ptr.To(toLevel(conf.Log.HTTPLevel)),
		},
		GeoLocation: &configv1.GeoLocation{
			Enabled:   &conf.GeoLocation.Enabled,
			CachePath: &conf.GeoLocation.CachePath,
			Token:     &conf.GeoLocation.Token,
		},
		Ssh: &configv1.SSH{
			Enabled:         &conf.SSH.Enabled,
			Username:        &conf.SSH.Username,
			Port:            ptr.To(int32(conf.SSH.Port)),
			PrivateKeyPath:  &conf.SSH.PrivateKeyPath,
			HostKeyStrategy: ptr.To(configv1.HostKeyStrategy(conf.SSH.HostKeyStrategy)),
			Password:        &conf.SSH.Password,
			UpdateInterval:  ptr.To(int32(conf.SSH.UpdateInterval)),
			Timeout:         ptr.To(int32(conf.SSH.Timeout)),
			DemoPathFmt:     &conf.SSH.DemoPathFmt,
			StacPath:        &conf.SSH.StacPathFmt,
		},
		Network: &configv1.Network{
			SdrEnabled:    &conf.Network.SDREnabled,
			SdrDnsEnabled: &conf.Network.SDRDNSEnabled,
			CfKey:         &conf.Network.CFKey,
			CfEmail:       &conf.Network.CFEmail,
			CfZoneId:      &conf.Network.CFZoneID,
		},
		LocalStore: &configv1.LocalStore{
			PathRoot: &conf.LocalStore.PathRoot,
		},
		Exports: &configv1.Exports{
			BdEnabled:      &conf.Exports.BDEnabled,
			ValveEnabled:   &conf.Exports.ValveEnabled,
			AuthorizedKeys: strings.Split(conf.Exports.AuthorizedKeys, ","),
		},
		Anticheat: &configv1.Anticheat{
			Enabled:               &conf.Anticheat.Enabled,
			Action:                ptr.To(toAction(conf.Anticheat.Action)),
			Duration:              ptr.To(int32(conf.Anticheat.Duration)),
			MaxAimSnaps:           ptr.To(int32(conf.Anticheat.MaxAimSnap)),
			MaxPsilent:            ptr.To(int32(conf.Anticheat.MaxPsilent)),
			MaxBhop:               ptr.To(int32(conf.Anticheat.MaxBhop)),
			MaxFakeAng:            ptr.To(int32(conf.Anticheat.MaxFakeAng)),
			MaxCmdNum:             ptr.To(int32(conf.Anticheat.MaxCmdNum)),
			MaxTooManyConnections: ptr.To(int32(conf.Anticheat.MaxTooManyConnections)),
			MaxOobVar:             ptr.To(int32(conf.Anticheat.MaxOOBVar)),
			MaxInvalidUserCmd:     ptr.To(int32(conf.Anticheat.MaxInvalidUserCmd)),
		},
	}
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
