package config

import (
	"context"
	"strings"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/anticheat"
	"github.com/leighmacdonald/gbans/internal/asset"
	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/chat"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/log"
	"github.com/leighmacdonald/gbans/internal/network"
	"github.com/leighmacdonald/gbans/internal/network/ip2location"
	"github.com/leighmacdonald/gbans/internal/network/scp"
	"github.com/leighmacdonald/gbans/internal/patreon"
	configv1 "github.com/leighmacdonald/gbans/internal/rpc/config/v1"
	"github.com/leighmacdonald/gbans/internal/rpc/config/v1/configv1connect"
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
		SiteName:           conf.General.SiteName,
		AssetUrl:           conf.General.AssetURL,
		Favicon:            conf.General.FaviconURL(),
		LinkId:             conf.Discord.LinkID,
		AppVersion:         r.version,
		DocumentPolicy:     "",
		SentryDsnWeb:       conf.General.SentryDSNWeb,
		SiteDescription:    conf.General.SiteDescription,
		PatreonClientId:    conf.Patreon.ClientID,
		DiscordClientId:    conf.Discord.AppID,
		DiscordEnabled:     conf.Discord.IntegrationsEnabled && conf.Discord.Enabled,
		PatreonEnabled:     conf.Patreon.IntegrationsEnabled && conf.Patreon.Enabled,
		DefaultRoute:       conf.General.DefaultRoute,
		NewsEnabled:        conf.General.NewsEnabled,
		ForumsEnabled:      conf.General.ForumsEnabled,
		ContestsEnabled:    conf.General.ContestsEnabled,
		WikiEnabled:        conf.General.WikiEnabled,
		StatsEnabled:       conf.General.StatsEnabled,
		ServersEnabled:     conf.General.ServersEnabled,
		ReportsEnabled:     conf.General.ReportsEnabled,
		ChatlogsEnabled:    conf.General.ChatlogsEnabled,
		DemosEnabled:       conf.General.DemosEnabled,
		SpeedrunsEnabled:   conf.General.SpeedrunsEnabled,
		PlayerqueueEnabled: conf.General.PlayerqueueEnabled,
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
		General: &configv1.General{
			SiteName:           c.General.SiteName,
			SiteDescription:    c.General.SiteDescription,
			Mode:               toRunMode(c.General.Mode),
			FileServeMode:      toServeMode(c.General.FileServeMode),
			SrcdsLogAddr:       c.General.SrcdsLogAddr,
			AssetUrl:           c.General.AssetURL,
			Favicon:            c.General.Favicon,
			WikiEnabled:        c.General.WikiEnabled,
			DefaultRoute:       c.General.DefaultRoute,
			NewsEnabled:        c.General.NewsEnabled,
			ForumsEnabled:      c.General.ForumsEnabled,
			ContestsEnabled:    c.General.ContestsEnabled,
			StatsEnabled:       c.General.StatsEnabled,
			ServersEnabled:     c.General.ServersEnabled,
			ReportsEnabled:     c.General.ReportsEnabled,
			ChatlogsEnabled:    c.General.ChatlogsEnabled,
			DemosEnabled:       c.General.DemosEnabled,
			SpeedrunsEnabled:   c.General.SpeedrunsEnabled,
			PlayerqueueEnabled: c.General.PlayerqueueEnabled,
			SentryDsn:          c.General.SentryDSN,
			SentryDsnWeb:       c.General.SentryDSNWeb,
		},
		Discord: &configv1.Discord{
			Token:                   c.Discord.Token,
			Enabled:                 c.Discord.Enabled,
			BotEnabled:              c.Discord.BotEnabled,
			IntegrationsEnabled:     c.Discord.IntegrationsEnabled,
			AppId:                   c.Discord.AppID,
			AppSecret:               c.Discord.AppSecret,
			LinkId:                  c.Discord.LinkID,
			GuildId:                 c.Discord.GuildID,
			PublicLogChannelEnable:  c.Discord.PublicLogChannelEnable,
			LogChannelId:            c.Discord.LogChannelID,
			PublicMatchLogChannelId: c.Discord.PublicMatchLogChannelID,
			VoteLogChannelId:        c.Discord.VoteLogChannelID,
			AppealLogChannelId:      c.Discord.AppealLogChannelID,
			BanLogChannelId:         c.Discord.BanLogChannelID,
			ForumLogChannelId:       c.Discord.ForumLogChannelID,
			KickLogChannelId:        c.Discord.KickLogChannelID,
			PlayerqueueChannelId:    c.Discord.PlayerqueueChannelID,
			ModPingRoleId:           c.Discord.ModPingRoleID,
			AnticheatChannelId:      c.Discord.AnticheatChannelID,
			SeedChannelId:           c.Discord.SeedChannelID,
			WordFilterLogChannelId:  c.Discord.WordFilterLogChannelID,
			ChatLogChannelId:        c.Discord.ChatLogChannelID,
		},
		Patreon: &configv1.Patreon{
			Enabled:             c.Patreon.Enabled,
			IntegrationsEnabled: c.Patreon.IntegrationsEnabled,
			ClientId:            c.Patreon.ClientID,
			ClientSecret:        c.Patreon.ClientSecret,
			CreatorAccessToken:  c.Patreon.CreatorAccessToken,
			CreatorRefreshToken: c.Patreon.CreatorRefreshToken,
		},
		Debug: &configv1.Debug{
			SkipOpenIdValidation: c.Debug.SkipOpenIDValidation,
			AddRconLogAddress:    c.Debug.AddRCONLogAddress,
		},
		Demo: &configv1.Demo{
			CleanupEnabled: c.Demo.DemoCleanupEnabled,
			Strategy:       toDemoStrategy(c.Demo.DemoCleanupStrategy),
			CleanupMinPct:  c.Demo.DemoCleanupMinPct,
			CleanupMount:   c.Demo.DemoCleanupMount,
			CountLimit:     int64(c.Demo.DemoCountLimit),
			ParserUrl:      c.Demo.DemoParserURL,
		},
		Filters: &configv1.Filters{
			Enabled:        c.Filters.Enabled,
			WarningTimeout: int32(c.Filters.WarningTimeout),
			WarningLimit:   int32(c.Filters.WarningLimit),
			Dry:            c.Filters.Dry,
			PingDiscord:    c.Filters.PingDiscord,
			MaxWeight:      int32(c.Filters.MaxWeight),
			CheckTimeout:   int32(c.Filters.CheckTimeout),
			MatchTimeout:   int32(c.Filters.MatchTimeout),
		},
		Log: &configv1.Log{
			Level:           toLevel(c.Log.Level),
			File:            &c.Log.File,
			HttpEnabled:     c.Log.HTTPEnabled,
			HttpOtelEnabled: c.Log.HTTPOtelEnabled,
			HttpLevel:       toLevel(c.Log.HTTPLevel),
		},
		GeoLocation: &configv1.GeoLocation{
			Enabled:   c.GeoLocation.Enabled,
			CachePath: c.GeoLocation.CachePath,
			Token:     c.GeoLocation.Token,
		},
		Ssh: &configv1.SSH{
			Enabled:         c.SSH.Enabled,
			Username:        c.SSH.Username,
			Port:            int32(c.SSH.Port),
			PrivateKeyPath:  c.SSH.PrivateKeyPath,
			HostKeyStrategy: configv1.HostKeyStrategy(c.SSH.HostKeyStrategy),
			Password:        c.SSH.Password,
			UpdateInterval:  int32(c.SSH.UpdateInterval),
			Timeout:         int32(c.SSH.Timeout),
			DemoPathFmt:     c.SSH.DemoPathFmt,
			StacPath:        c.SSH.StacPathFmt,
		},
		Network: &configv1.Network{
			SdrEnabled:    c.Network.SDREnabled,
			SdrDnsEnabled: c.Network.SDRDNSEnabled,
			CfKey:         c.Network.CFKey,
			CfEmail:       c.Network.CFEmail,
			CfZoneId:      c.Network.CFZoneID,
		},
		LocalStore: &configv1.LocalStore{
			PathRoot: c.LocalStore.PathRoot,
		},
		Exports: &configv1.Exports{
			BdEnabled:      c.Exports.BDEnabled,
			ValveEnabled:   c.Exports.ValveEnabled,
			AuthorizedKeys: strings.Split(c.Exports.AuthorizedKeys, ","),
		},
		Anticheat: &configv1.Anticheat{
			Enabled:               c.Anticheat.Enabled,
			Action:                toAction(c.Anticheat.Action),
			Duration:              int32(c.Anticheat.Duration),
			MaxAimSnaps:           int32(c.Anticheat.MaxAimSnap),
			MaxPsilent:            int32(c.Anticheat.MaxPsilent),
			MaxBhop:               int32(c.Anticheat.MaxBhop),
			MaxFakeAng:            int32(c.Anticheat.MaxFakeAng),
			MaxCmdNum:             int32(c.Anticheat.MaxCmdNum),
			MaxTooManyConnections: int32(c.Anticheat.MaxTooManyConnections),
			MaxOobVar:             int32(c.Anticheat.MaxOOBVar),
			MaxInvalidUserCmd:     int32(c.Anticheat.MaxInvalidUserCmd),
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
			SiteName:           request.General.SiteName,
			SiteDescription:    request.General.SiteDescription,
			Mode:               RunMode(request.General.Mode.String()),
			FileServeMode:      FileServeMode(request.General.FileServeMode.String()),
			SrcdsLogAddr:       request.General.SrcdsLogAddr,
			AssetURL:           request.General.AssetUrl,
			Favicon:            request.General.Favicon,
			DefaultRoute:       request.General.DefaultRoute,
			NewsEnabled:        request.General.NewsEnabled,
			ForumsEnabled:      request.General.ForumsEnabled,
			ContestsEnabled:    request.General.ContestsEnabled,
			WikiEnabled:        request.General.WikiEnabled,
			StatsEnabled:       request.General.StatsEnabled,
			ServersEnabled:     request.General.ServersEnabled,
			ReportsEnabled:     request.General.ReportsEnabled,
			ChatlogsEnabled:    request.General.ChatlogsEnabled,
			DemosEnabled:       request.General.DemosEnabled,
			SpeedrunsEnabled:   request.General.SpeedrunsEnabled,
			PlayerqueueEnabled: request.General.PlayerqueueEnabled,
			SentryDSN:          request.General.SentryDsn,
			SentryDSNWeb:       request.General.SentryDsnWeb,
		},
		Debug: Debug{
			SkipOpenIDValidation: request.Debug.SkipOpenIdValidation,
			AddRCONLogAddress:    request.Debug.AddRconLogAddress,
		},
		Demo: servers.DemoConfig{
			DemoCleanupEnabled:  request.Demo.CleanupEnabled,
			DemoCleanupStrategy: servers.DemoStrategy(request.Demo.Strategy.String()),
			DemoCleanupMinPct:   request.Demo.CleanupMinPct,
			DemoCleanupMount:    request.Demo.CleanupMount,
			DemoCountLimit:      uint64(request.Demo.CountLimit),
			DemoParserURL:       request.Demo.ParserUrl,
		},
		Filters: chat.Config{
			Enabled:        request.Filters.Enabled,
			WarningTimeout: int(request.Filters.WarningTimeout),
			WarningLimit:   int(request.Filters.WarningLimit),
			Dry:            request.Filters.Dry,
			PingDiscord:    request.Filters.PingDiscord,
			MaxWeight:      int(request.Filters.MaxWeight),
			CheckTimeout:   int(request.Filters.CheckTimeout),
			MatchTimeout:   int(request.Filters.MatchTimeout),
		},
		Discord: discord.Config{
			Enabled:                 request.Discord.Enabled,
			BotEnabled:              request.Discord.BotEnabled,
			IntegrationsEnabled:     request.Discord.IntegrationsEnabled,
			AppID:                   request.Discord.AppId,
			AppSecret:               request.Discord.AppSecret,
			LinkID:                  request.Discord.LinkId,
			Token:                   request.Discord.Token,
			GuildID:                 request.Discord.GuildId,
			PublicLogChannelEnable:  request.Discord.PublicLogChannelEnable,
			LogChannelID:            request.Discord.LogChannelId,
			PublicLogChannelID:      request.Discord.PublicMatchLogChannelId,
			PublicMatchLogChannelID: request.Discord.PublicMatchLogChannelId,
			VoteLogChannelID:        request.Discord.VoteLogChannelId,
			AppealLogChannelID:      request.Discord.AppealLogChannelId,
			BanLogChannelID:         request.Discord.BanLogChannelId,
			ForumLogChannelID:       request.Discord.ForumLogChannelId,
			KickLogChannelID:        request.Discord.KickLogChannelId,
			PlayerqueueChannelID:    request.Discord.PlayerqueueChannelId,
			ModPingRoleID:           request.Discord.ModPingRoleId,
			AnticheatChannelID:      request.Discord.AnticheatChannelId,
			SeedChannelID:           request.Discord.SeedChannelId,
			WordFilterLogChannelID:  request.Discord.WordFilterLogChannelId,
			ChatLogChannelID:        request.Discord.ChatLogChannelId,
		},
		Clientprefs: sourcemod.Config{
			// CenterProjectiles: request.,
		},
		Log: log.Config{
			Level:           log.Level(request.Log.Level.String()),
			File:            *request.Log.File,
			HTTPEnabled:     request.Log.HttpEnabled,
			HTTPOtelEnabled: request.Log.HttpOtelEnabled,
			HTTPLevel:       log.Level(request.Log.HttpLevel.String()),
		},
		GeoLocation: ip2location.Config{
			Enabled:   request.GeoLocation.Enabled,
			CachePath: request.GeoLocation.CachePath,
			Token:     request.GeoLocation.Token,
		},
		Patreon: patreon.Config{
			Enabled:             request.Patreon.Enabled,
			IntegrationsEnabled: request.Patreon.IntegrationsEnabled,
			ClientID:            request.Patreon.ClientId,
			ClientSecret:        request.Patreon.ClientSecret,
			CreatorAccessToken:  request.Patreon.CreatorAccessToken,
			CreatorRefreshToken: request.Patreon.CreatorRefreshToken,
		},
		SSH: scp.Config{
			Enabled:         request.Ssh.Enabled,
			Username:        request.Ssh.Username,
			Port:            int(request.Ssh.Port),
			PrivateKeyPath:  request.Ssh.PrivateKeyPath,
			HostKeyStrategy: scp.HostKeyStrategy(request.Ssh.HostKeyStrategy),
			Password:        request.Ssh.Password,
			UpdateInterval:  int(request.Ssh.UpdateInterval),
			Timeout:         int(request.Ssh.Timeout),
			DemoPathFmt:     request.Ssh.DemoPathFmt,
			StacPathFmt:     request.Ssh.StacPath,
		},
		Network: network.Config{
			SDREnabled:    request.Network.SdrEnabled,
			SDRDNSEnabled: request.Network.SdrDnsEnabled,
			CFKey:         request.Network.CfKey,
			CFEmail:       request.Network.CfEmail,
			CFZoneID:      request.Network.CfZoneId,
		},
		LocalStore: asset.Config{
			PathRoot: request.LocalStore.PathRoot,
		},
		Exports: ban.Config{
			BDEnabled:      request.Exports.BdEnabled,
			ValveEnabled:   request.Exports.ValveEnabled,
			AuthorizedKeys: strings.Join(request.Exports.AuthorizedKeys, ","),
		},
		Anticheat: anticheat.Config{
			Enabled:               request.Anticheat.Enabled,
			Action:                anticheat.Action(request.Anticheat.Action.String()),
			Duration:              int(request.Anticheat.Duration),
			MaxAimSnap:            int(request.Anticheat.MaxAimSnaps),
			MaxPsilent:            int(request.Anticheat.MaxPsilent),
			MaxBhop:               int(request.Anticheat.MaxBhop),
			MaxFakeAng:            int(request.Anticheat.MaxFakeAng),
			MaxCmdNum:             int(request.Anticheat.MaxCmdNum),
			MaxTooManyConnections: int(request.Anticheat.MaxTooManyConnections),
			MaxCheatCvar:          int(request.Anticheat.MaxCheatCvar),
			MaxOOBVar:             int(request.Anticheat.MaxOobVar),
			MaxInvalidUserCmd:     int(request.Anticheat.MaxInvalidUserCmd),
		},
	}
	if errWrite := r.Configuration.Write(ctx, conf); errWrite != nil {
		return nil, connect.NewError(connect.CodeUnknown, errWrite)
	}

	return nil, nil
}
