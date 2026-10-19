package config

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/leighmacdonald/gbans/internal/anticheat"
	"github.com/leighmacdonald/gbans/internal/asset"
	"github.com/leighmacdonald/gbans/internal/ban"
	"github.com/leighmacdonald/gbans/internal/chat"
	configv1 "github.com/leighmacdonald/gbans/internal/config/v1"
	"github.com/leighmacdonald/gbans/internal/config/v1/configv1connect"
	"github.com/leighmacdonald/gbans/internal/demo"
	"github.com/leighmacdonald/gbans/internal/discord"
	"github.com/leighmacdonald/gbans/internal/log"
	"github.com/leighmacdonald/gbans/internal/network"
	"github.com/leighmacdonald/gbans/internal/network/ip2location"
	"github.com/leighmacdonald/gbans/internal/network/scp"
	"github.com/leighmacdonald/gbans/internal/opengraph"
	"github.com/leighmacdonald/gbans/internal/patreon"
	rolesv1 "github.com/leighmacdonald/gbans/internal/roles/v1"
	"github.com/leighmacdonald/gbans/internal/rpc"
	"github.com/leighmacdonald/gbans/internal/sourcemod"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service struct {
	// configv1connect.UnimplementedConfigServiceHandler

	config  *Configuration
	version string

	changelogMu      sync.Mutex
	changelogCache   []*configv1.GithubRelease
	changelogFetched time.Time
}

func NewService(conf *Configuration, version string, openGraph *opengraph.Repository, roleAuth *rpc.RoleAuth, authMiddleware *rpc.Middleware, options ...connect.HandlerOption) rpc.Service {
	pattern, handler := configv1connect.NewConfigServiceHandler(&Service{
		config:  conf,
		version: version,
	}, options...)

	openGraph.Register("/admin/settings", opengraph.Static("System Settings", "Edit core system settings."))
	openGraph.Register("/changelog", opengraph.Static("Changelog", "Recent updates and changes."))
	openGraph.Register("/privacy-policy", opengraph.Static("Privacy Policy", "How we handle your data."))

	authMiddleware.UserRoute(configv1connect.ConfigServiceGetProcedure, roleAuth.WithOneOf(rolesv1.Permission_PERMISSION_CONFIG_READ))
	authMiddleware.UserRoute(configv1connect.ConfigServiceUpdateProcedure, roleAuth.WithOneOf(rolesv1.Permission_PERMISSION_CONFIG_WRITE))

	return rpc.Service{
		Pattern: pattern,
		Handler: handler,
	}
}

const (
	// changelogRepoOwner/Name must match the releases linked in the footer.
	changelogRepoOwner = "leighmacdonald"
	changelogRepoName  = "gbans"
	// changelogMaxReleases bounds the payload; the page renders newest first.
	changelogMaxReleases = 10
	// changelogCacheTTL keeps us far under GitHub's unauthenticated rate
	// limits (60 req/hour) even with many viewers.
	changelogCacheTTL = time.Minute * 30
)

// githubReleaseJSON mirrors the GitHub releases API field names.
type githubReleaseJSON struct {
	ID          int64  `json:"id"`
	TagName     string `json:"tag_name"` //nolint:tagliatelle // GitHub API snake_case.
	Name        string `json:"name"`
	Body        string `json:"body"`
	Draft       bool   `json:"draft"`
	Prerelease  bool   `json:"prerelease"`
	CreatedAt   string `json:"created_at"`   //nolint:tagliatelle // GitHub API snake_case.
	PublishedAt string `json:"published_at"` //nolint:tagliatelle // GitHub API snake_case.
}

func fetchGithubReleases(ctx context.Context, owner string, repo string, limit int) ([]*configv1.GithubRelease, error) {
	req, errReq := http.NewRequestWithContext(ctx,
		http.MethodGet,
		fmt.Sprintf("https://api.github.com/repos/%s/%s/releases?per_page=%d", owner, repo, limit),
		nil,
	)
	if errReq != nil {
		return nil, fmt.Errorf("create github releases request: %w", errReq)
	}

	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "gbans-changelog")
	req.Header.Set("X-Github-Api-Version", "2022-11-28")

	client := &http.Client{Timeout: time.Second * 10}

	resp, errResp := client.Do(req)
	if errResp != nil {
		return nil, fmt.Errorf("fetch github releases: %w", errResp)
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github releases status: %d", resp.StatusCode) //nolint:err113
	}

	body, errBody := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if errBody != nil {
		return nil, fmt.Errorf("read github releases body: %w", errBody)
	}

	var raw []githubReleaseJSON
	if errDecode := json.Unmarshal(body, &raw); errDecode != nil {
		return nil, fmt.Errorf("decode github releases: %w", errDecode)
	}

	releases := make([]*configv1.GithubRelease, 0, len(raw))
	for _, item := range raw {
		name := item.Name
		if name == "" {
			name = item.TagName
		}

		createdAt, errCreated := time.Parse(time.RFC3339, item.CreatedAt)
		if errCreated != nil {
			continue
		}

		publishedAt, errPublished := time.Parse(time.RFC3339, item.PublishedAt)
		if errPublished != nil {
			publishedAt = createdAt
		}

		id := int32(item.ID) //nolint:gosec

		releases = append(
			releases, &configv1.GithubRelease{
				Id:          &id,
				TagName:     &item.TagName,
				Name:        &name,
				Body:        &item.Body,
				Draft:       &item.Draft,
				Prerelease:  &item.Prerelease,
				CreatedAt:   timestamppb.New(createdAt),
				PublishedAt: timestamppb.New(publishedAt),
			},
		)
	}

	return releases, nil
}

func (r *Service) Changelog(ctx context.Context, _ *emptypb.Empty) (*configv1.ChangelogResponse, error) {
	r.changelogMu.Lock()
	defer r.changelogMu.Unlock()

	if time.Since(r.changelogFetched) < changelogCacheTTL && r.changelogCache != nil {
		return &configv1.ChangelogResponse{Changelog: r.changelogCache}, nil
	}

	releases, errFetch := fetchGithubReleases(ctx, changelogRepoOwner, changelogRepoName, changelogMaxReleases)
	if errFetch != nil {
		// Serve stale cache rather than breaking the page on transient
		// GitHub outages.
		if r.changelogCache != nil {
			slog.Warn("Serving stale changelog cache", slog.String("error", errFetch.Error()))

			return &configv1.ChangelogResponse{Changelog: r.changelogCache}, nil
		}

		return nil, connect.NewError(connect.CodeInternal, errFetch)
	}

	r.changelogCache = releases
	r.changelogFetched = time.Now()

	return &configv1.ChangelogResponse{Changelog: releases}, nil
}

func (r *Service) Info(_ context.Context, _ *emptypb.Empty) (*configv1.InfoResponse, error) {
	conf := r.config.Config()
	conf.General.RLock()
	conf.Discord.RLock()
	conf.Patreon.RLock()
	defer conf.General.RUnlock()
	defer conf.Discord.RUnlock()
	defer conf.Patreon.RUnlock()

	resp := configv1.InfoResponse{
		SiteName:         &conf.General.SiteName,
		AssetUrl:         &conf.General.AssetURL,
		Favicon:          new(conf.General.FaviconURL()),
		BannerImage:      new(conf.General.BannerImageURL()),
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
	conf := configFromUpdate(request.GetConfig(), r.config.Config())
	if errWrite := r.config.Write(ctx, conf); errWrite != nil {
		return nil, connect.NewError(connect.CodeUnknown, errWrite)
	}

	return &configv1.UpdateResponse{Config: toConfig(conf)}, nil
}

// configFromUpdate maps the api representation onto the domain config.
// PlayerqueueEnabled and CenterProjectiles are not managed through the api;
// the current values are carried over so a save does not wipe them.
func configFromUpdate(inCfg *configv1.Config, current Config) Config {
	inGeneral := inCfg.GetGeneral()
	inDebug := inCfg.GetDebug()
	inAC := inCfg.GetAnticheat()
	inExports := inCfg.GetExports()
	inLocalStore := inCfg.GetLocalStore()
	inDemo := inCfg.GetDemo()
	inNetwork := inCfg.GetNetwork()
	inDiscord := inCfg.GetDiscord()
	inSSH := inCfg.GetSsh()
	inPatreon := inCfg.GetPatreon()
	inGeo := inCfg.GetGeoLocation()
	inLog := inCfg.GetLog()
	inFilters := inCfg.GetFilters()

	conf := Config{
		General: &General{
			SiteName:           inGeneral.GetSiteName(),
			SiteDescription:    inGeneral.GetSiteDescription(),
			Mode:               fromRunMode(inGeneral.GetMode()),
			FileServeMode:      fromServeMode(inGeneral.GetFileServeMode()),
			SrcdsLogAddr:       inGeneral.GetSrcdsLogAddr(),
			AssetURL:           inGeneral.GetAssetUrl(),
			Favicon:            inGeneral.GetFavicon(),
			BannerImage:        inGeneral.GetBannerImage(),
			DefaultRoute:       inGeneral.GetDefaultRoute(),
			NewsEnabled:        inGeneral.GetNewsEnabled(),
			ForumsEnabled:      inGeneral.GetForumsEnabled(),
			ContestsEnabled:    inGeneral.GetContestsEnabled(),
			WikiEnabled:        inGeneral.GetWikiEnabled(),
			StatsEnabled:       inGeneral.GetStatsEnabled(),
			ServersEnabled:     inGeneral.GetServersEnabled(),
			ReportsEnabled:     inGeneral.GetReportsEnabled(),
			ChatlogsEnabled:    inGeneral.GetChatlogsEnabled(),
			DemosEnabled:       inGeneral.GetDemosEnabled(),
			PlayerqueueEnabled: current.General.PlayerqueueEnabled,
			SpeedrunsEnabled:   inGeneral.GetSpeedrunsEnabled(),
			MGEEnabled:         inGeneral.GetMgeEnabled(),
			SentryDSN:          inGeneral.GetSentryDsn(),
			SentryDSNWeb:       inGeneral.GetSentryDsnWeb(),
		},
		Debug: &Debug{
			SkipOpenIDValidation: inDebug.GetSkipOpenIdValidation(),
			AddRCONLogAddress:    inDebug.GetAddRconLogAddress(),
		},
		Demo: &demo.Config{
			DemoCleanupEnabled:  inDemo.GetCleanupEnabled(),
			DemoCleanupStrategy: fromDemoStrategy(inDemo.GetStrategy()),
			DemoCleanupMinPct:   inDemo.GetCleanupMinPct(),
			DemoCleanupMount:    inDemo.GetCleanupMount(),
			DemoCountLimit:      uint64(inDemo.GetCountLimit()), //nolint:gosec
			DemoParserURL:       inDemo.GetParserUrl(),
		},
		Filters: &chat.Config{
			Enabled:        inFilters.GetEnabled(),
			WarningTimeout: inFilters.GetWarningTimeout(),
			WarningLimit:   inFilters.GetWarningLimit(),
			Dry:            inFilters.GetDry(),
			PingDiscord:    inFilters.GetPingDiscord(),
			MaxWeight:      inFilters.GetMaxWeight(),
			CheckTimeout:   inFilters.GetCheckTimeout(),
			MatchTimeout:   inFilters.GetMatchTimeout(),
		},
		Discord: &discord.Config{
			Enabled:                 inDiscord.GetEnabled(),
			BotEnabled:              inDiscord.GetBotEnabled(),
			IntegrationsEnabled:     inDiscord.GetIntegrationsEnabled(),
			AppID:                   inDiscord.GetAppId(),
			AppSecret:               inDiscord.GetAppSecret(),
			LinkID:                  inDiscord.GetLinkId(),
			Token:                   inDiscord.GetToken(),
			GuildID:                 inDiscord.GetGuildId(),
			PublicLogChannelEnable:  inDiscord.GetPublicLogChannelEnable(),
			LogChannelID:            inDiscord.GetLogChannelId(),
			PublicLogChannelID:      inDiscord.GetPublicLogChannelId(),
			PublicMatchLogChannelID: inDiscord.GetPublicMatchLogChannelId(),
			VoteLogChannelID:        inDiscord.GetVoteLogChannelId(),
			AppealLogChannelID:      inDiscord.GetAppealLogChannelId(),
			BanLogChannelID:         inDiscord.GetBanLogChannelId(),
			ForumLogChannelID:       inDiscord.GetForumLogChannelId(),
			KickLogChannelID:        inDiscord.GetKickLogChannelId(),
			ModPingRoleID:           inDiscord.GetModPingRoleId(),
			AnticheatChannelID:      inDiscord.GetAnticheatChannelId(),
			SeedChannelID:           inDiscord.GetSeedChannelId(),
			WordFilterLogChannelID:  inDiscord.GetWordFilterLogChannelId(),
			ChatLogChannelID:        inDiscord.GetChatLogChannelId(),
		},
		Clientprefs: &sourcemod.Config{
			CenterProjectiles: current.Clientprefs.CenterProjectiles,
		},
		Log: &log.Config{
			Level:           fromLevel(inLog.GetLevel()),
			File:            inLog.GetFile(),
			HTTPEnabled:     inLog.GetHttpEnabled(),
			HTTPOtelEnabled: inLog.GetHttpOtelEnabled(),
			HTTPLevel:       fromLevel(inLog.GetHttpLevel()),
		},
		GeoLocation: &ip2location.Config{
			Enabled:            inGeo.GetEnabled(),
			CachePath:          inGeo.GetCachePath(),
			Token:              inGeo.GetToken(),
			UpdateIntervalDays: int(inGeo.GetUpdateIntervalDays()), //nolint:gosec
		},
		Patreon: &patreon.Config{
			Enabled:             inPatreon.GetEnabled(),
			IntegrationsEnabled: inPatreon.GetIntegrationsEnabled(),
			ClientID:            inPatreon.GetClientId(),
			ClientSecret:        inPatreon.GetClientSecret(),
			CreatorAccessToken:  inPatreon.GetCreatorAccessToken(),
			CreatorRefreshToken: inPatreon.GetCreatorRefreshToken(),
		},
		SSH: &scp.Config{
			Enabled:         inSSH.GetEnabled(),
			Username:        inSSH.GetUsername(),
			Port:            uint16(inSSH.GetPort()), //nolint:gosec
			PrivateKeyPath:  inSSH.GetPrivateKeyPath(),
			HostKeyStrategy: scp.HostKeyStrategy(inSSH.GetHostKeyStrategy()),
			Password:        inSSH.GetPassword(),
			UpdateInterval:  inSSH.GetUpdateInterval(),
			Timeout:         inSSH.GetTimeout(),
			DemoPathFmt:     inSSH.GetDemoPathFmt(),
			StacPathFmt:     inSSH.GetStacPathFmt(),
		},
		Network: &network.Config{
			SDREnabled: inNetwork.GetSdrEnabled(),
		},
		LocalStore: &asset.Config{
			PathRoot: inLocalStore.GetPathRoot(),
		},
		Exports: &ban.Config{
			BDEnabled:      inExports.GetBdEnabled(),
			ValveEnabled:   inExports.GetValveEnabled(),
			AuthorizedKeys: strings.Join(inExports.GetAuthorizedKeys(), ","),
		},
		Anticheat: &anticheat.Config{
			Enabled:               inAC.GetEnabled(),
			Action:                fromAction(inAC.GetAction()),
			Duration:              inAC.GetDuration(),
			MaxAimSnap:            inAC.GetMaxAimSnaps(),
			MaxPsilent:            inAC.GetMaxPsilent(),
			MaxBhop:               inAC.GetMaxBhop(),
			MaxFakeAng:            inAC.GetMaxFakeAng(),
			MaxCmdNum:             inAC.GetMaxCmdNum(),
			MaxTooManyConnections: inAC.GetMaxTooManyConnections(),
			MaxCheatCvar:          inAC.GetMaxCheatCvar(),
			MaxOOBVar:             inAC.GetMaxOobVar(),
			MaxInvalidUserCmd:     inAC.GetMaxInvalidUserCmd(),
		},
	}

	return conf
}

func toConfig(conf Config) *configv1.Config {
	return &configv1.Config{
		General: &configv1.General{
			SiteName:         &conf.General.SiteName,
			SiteDescription:  &conf.General.SiteDescription,
			Mode:             new(toRunMode(conf.General.Mode)),
			FileServeMode:    new(toServeMode(conf.General.FileServeMode)),
			SrcdsLogAddr:     &conf.General.SrcdsLogAddr,
			AssetUrl:         &conf.General.AssetURL,
			Favicon:          &conf.General.Favicon,
			BannerImage:      &conf.General.BannerImage,
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
			MgeEnabled:       &conf.General.MGEEnabled,
			SentryDsn:        &conf.General.SentryDSN,
			SentryDsnWeb:     &conf.General.SentryDSNWeb,
		},
		Discord: &configv1.Discord{
			Enabled:                 &conf.Discord.Enabled,
			BotEnabled:              &conf.Discord.BotEnabled,
			IntegrationsEnabled:     &conf.Discord.IntegrationsEnabled,
			AppId:                   &conf.Discord.AppID,
			LinkId:                  &conf.Discord.LinkID,
			GuildId:                 &conf.Discord.GuildID,
			AppSecret:               &conf.Discord.AppSecret,
			Token:                   &conf.Discord.Token,
			PublicLogChannelEnable:  &conf.Discord.PublicLogChannelEnable,
			LogChannelId:            &conf.Discord.LogChannelID,
			PublicLogChannelId:      &conf.Discord.PublicLogChannelID,
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
			Strategy:       new(toDemoStrategy(conf.Demo.DemoCleanupStrategy)),
			CleanupMinPct:  &conf.Demo.DemoCleanupMinPct,
			CleanupMount:   &conf.Demo.DemoCleanupMount,
			CountLimit:     new(int64(conf.Demo.DemoCountLimit)), //nolint:gosec
			ParserUrl:      &conf.Demo.DemoParserURL,
		},
		Filters: &configv1.Filters{
			Enabled:        &conf.Filters.Enabled,
			WarningTimeout: new(conf.Filters.WarningTimeout),
			WarningLimit:   new(conf.Filters.WarningLimit),
			Dry:            &conf.Filters.Dry,
			PingDiscord:    &conf.Filters.PingDiscord,
			MaxWeight:      new(conf.Filters.MaxWeight),
			CheckTimeout:   new(conf.Filters.CheckTimeout),
			MatchTimeout:   new(conf.Filters.MatchTimeout),
		},
		Log: &configv1.Log{
			Level:           new(toLevel(conf.Log.Level)),
			File:            &conf.Log.File,
			HttpEnabled:     &conf.Log.HTTPEnabled,
			HttpOtelEnabled: &conf.Log.HTTPOtelEnabled,
			HttpLevel:       new(toLevel(conf.Log.HTTPLevel)),
		},
		GeoLocation: &configv1.GeoLocation{
			Enabled:            &conf.GeoLocation.Enabled,
			CachePath:          &conf.GeoLocation.CachePath,
			Token:              &conf.GeoLocation.Token,
			UpdateIntervalDays: new(int32(conf.GeoLocation.UpdateIntervalDays)), //nolint:gosec
		},
		Ssh: &configv1.SSH{
			Enabled:         &conf.SSH.Enabled,
			Username:        &conf.SSH.Username,
			Port:            new(int32(conf.SSH.Port)),
			PrivateKeyPath:  &conf.SSH.PrivateKeyPath,
			Password:        &conf.SSH.Password,
			HostKeyStrategy: new(configv1.HostKeyStrategy(conf.SSH.HostKeyStrategy)), //nolint:gosec
			UpdateInterval:  new(conf.SSH.UpdateInterval),
			Timeout:         new(conf.SSH.Timeout),
			DemoPathFmt:     &conf.SSH.DemoPathFmt,
			StacPathFmt:     &conf.SSH.StacPathFmt,
		},
		Network: &configv1.Network{
			SdrEnabled: &conf.Network.SDREnabled,
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
			Action:                new(toAction(conf.Anticheat.Action)),
			Duration:              new(conf.Anticheat.Duration),
			MaxAimSnaps:           new(conf.Anticheat.MaxAimSnap),
			MaxPsilent:            new(conf.Anticheat.MaxPsilent),
			MaxBhop:               new(conf.Anticheat.MaxBhop),
			MaxFakeAng:            new(conf.Anticheat.MaxFakeAng),
			MaxCmdNum:             new(conf.Anticheat.MaxCmdNum),
			MaxTooManyConnections: new(conf.Anticheat.MaxTooManyConnections),
			MaxOobVar:             new(conf.Anticheat.MaxOOBVar),
			MaxInvalidUserCmd:     new(conf.Anticheat.MaxInvalidUserCmd),
			MaxCheatCvar:          new(conf.Anticheat.MaxCheatCvar),
		},
	}
}

func fromRunMode(mode configv1.RunMode) RunMode {
	switch mode {
	case configv1.RunMode_RUN_MODE_DEBUG:
		return DebugMode
	case configv1.RunMode_RUN_MODE_TEST:
		return TestMode
	case configv1.RunMode_RUN_MODE_RELEASE_UNSPECIFIED:
		fallthrough
	default:
		return ReleaseMode
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

func fromServeMode(mode configv1.FileServeMode) FileServeMode {
	switch mode {
	case configv1.FileServeMode_FILE_SERVE_MODE_LOCAL_UNSPECIFIED:
		fallthrough
	default:
		return LocalMode
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

func fromDemoStrategy(strategy configv1.DemoStrategy) demo.Strategy {
	switch strategy {
	case configv1.DemoStrategy_DEMO_STRATEGY_COUNT:
		return demo.DemoStrategyCount
	case configv1.DemoStrategy_DEMO_STRATEGY_PCTFREE_UNSPECIFIED:
		fallthrough
	default:
		return demo.DemoStrategyPctFree
	}
}

func toDemoStrategy(strategy demo.Strategy) configv1.DemoStrategy {
	switch strategy {
	case demo.DemoStrategyCount:
		return configv1.DemoStrategy_DEMO_STRATEGY_COUNT
	case demo.DemoStrategyPctFree:
		fallthrough
	default:
		return configv1.DemoStrategy_DEMO_STRATEGY_PCTFREE_UNSPECIFIED
	}
}

func fromLevel(level configv1.Level) log.Level {
	switch level {
	case configv1.Level_LEVEL_DEBUG:
		return log.Debug
	case configv1.Level_LEVEL_WARNING:
		return log.Warn
	case configv1.Level_LEVEL_INFO:
		return log.Info
	case configv1.Level_LEVEL_ERROR_UNSPECIFIED:
		fallthrough
	default:
		return log.Error
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

func fromAction(action configv1.Action) anticheat.Action {
	switch action {
	case configv1.Action_ACTION_BAN:
		return anticheat.ActionBan
	case configv1.Action_ACTION_GAG:
		return anticheat.ActionGag
	case configv1.Action_ACTION_KICK_UNSPECIFIED:
		fallthrough
	default:
		return anticheat.ActionKick
	}
}
