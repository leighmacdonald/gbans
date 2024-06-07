package config

import (
	"context"
	"errors"
	"sync"

	"github.com/leighmacdonald/gbans/internal/database"
	"github.com/leighmacdonald/gbans/internal/domain"
)

type configRepository struct {
	db   database.Database
	Conf domain.Config
	mu   sync.RWMutex
}

func NewConfigRepository(db database.Database) domain.ConfigRepository {
	return &configRepository{db: db, Conf: domain.Config{}}
}

func (c *configRepository) Config() domain.Config {
	c.mu.RLock()
	defer c.mu.RUnlock()

	return c.Conf
}

func (c *configRepository) Read(ctx context.Context) (domain.Config, error) {
	const query = `
		SELECT general_site_name, general_steam_key, general_mode, general_file_serve_mode, general_srcds_log_addr, general_asset_url,
		       general_default_route, general_news_enabled, general_forums_enabled, general_contests_enabled, general_wiki_enabled, 
		       general_stats_enabled, general_servers_enabled, general_reports_enabled,general_chatlogs_enabled, general_demos_enabled,
		       
		       filters_enabled, filters_dry, filters_ping_discord, filters_max_weight, filters_warning_timeout, filters_check_timeout, filters_match_timeout,
		       
		       demo_cleanup_enabled, demo_cleanup_strategy, demo_cleanup_min_pct, demo_cleanup_mount, demo_count_limit,
		       
		       patreon_enabled, patreon_client_id, patreon_client_secret, patreon_creator_access_token, patreon_creator_refresh_token, patreon_integrations_enabled,
		       
		       discord_enabled, discord_app_id, discord_app_secret, discord_link_id, discord_token, discord_guild_id, discord_log_channel_id,
		       discord_public_log_channel_enabled, discord_public_log_channel_id, discord_public_match_log_channel_id, discord_mod_ping_role_id,
		       discord_unregister_on_start, discord_bot_enabled, discord_integrations_enabled, discord_vote_log_channel_id ,discord_appeal_log_channel_id,
		       discord_ban_log_channel_id, discord_forum_log_channel_id, discord_word_filter_log_channel_id,
		       
		       logging_level, logging_file,
		       
		       sentry_sentry_dsn, sentry_sentry_dsn_web, sentry_sentry_trace, sentry_sentry_sample_rate,
		       
		       ip2location_enabled, ip2location_cache_path, ip2location_token,
		       
		       debug_skip_open_id_validation, debug_add_rcon_log_address,
		       
		       local_store_path_root,
		       
		       ssh_enabled, ssh_username, ssh_password, ssh_port, ssh_private_key_path, ssh_update_interval, ssh_timeout, ssh_demo_path_fmt,
		       
		       exports_bd_enabled, exports_valve_enabled, exports_authorized_keys
		 FROM config`

	var cfg domain.Config

	err := c.db.QueryRow(ctx, query).
		Scan(&cfg.General.SiteName, &cfg.General.SteamKey, &cfg.General.Mode, &cfg.General.FileServeMode, &cfg.General.SrcdsLogAddr, &cfg.General.AssetURL,
			&cfg.General.DefaultRoute, &cfg.General.NewsEnabled, &cfg.General.ForumsEnabled, &cfg.General.ContestsEnabled, &cfg.General.WikiEnabled,
			&cfg.General.StatsEnabled, &cfg.General.ServersEnabled, &cfg.General.ReportsEnabled, &cfg.General.ChatlogsEnabled, &cfg.General.DemosEnabled,
			&cfg.Filters.Enabled, &cfg.Filters.Dry, &cfg.Filters.PingDiscord, &cfg.Filters.MaxWeight, &cfg.Filters.WarningTimeout, &cfg.Filters.CheckTimeout, &cfg.Filters.MatchTimeout,
			&cfg.Demo.DemoCleanupEnabled, &cfg.Demo.DemoCleanupStrategy, &cfg.Demo.DemoCleanupMinPct, &cfg.Demo.DemoCleanupMount, &cfg.Demo.DemoCountLimit,
			&cfg.Patreon.Enabled, &cfg.Patreon.ClientID, &cfg.Patreon.ClientSecret, &cfg.Patreon.CreatorAccessToken, &cfg.Patreon.CreatorRefreshToken, &cfg.Patreon.IntegrationsEnabled,
			&cfg.Discord.Enabled, &cfg.Discord.AppID, &cfg.Discord.AppSecret, &cfg.Discord.LinkID, &cfg.Discord.Token, &cfg.Discord.GuildID, &cfg.Discord.LogChannelID,
			&cfg.Discord.PublicLogChannelEnable, &cfg.Discord.PublicLogChannelID, &cfg.Discord.PublicMatchLogChannelID, &cfg.Discord.ModPingRoleID,
			&cfg.Discord.UnregisterOnStart, &cfg.Discord.BotEnabled, &cfg.Discord.IntegrationsEnabled, &cfg.Discord.VoteLogChannelID, &cfg.Discord.AppealLogChannelID,
			&cfg.Discord.BanLogChannelID, &cfg.Discord.ForumLogChannelID, &cfg.Discord.WordFilterLogChannelID,
			&cfg.Log.Level, &cfg.Log.File,
			&cfg.Sentry.SentryDSN, &cfg.Sentry.SentryDSNWeb, &cfg.Sentry.SentryTrace, &cfg.Sentry.SentrySampleRate,
			&cfg.GeoLocation.Enabled, &cfg.GeoLocation.CachePath, &cfg.GeoLocation.Token,
			&cfg.Debug.SkipOpenIDValidation, &cfg.Debug.AddRCONLogAddress,
			&cfg.LocalStore.PathRoot,
			&cfg.SSH.Enabled, &cfg.SSH.Username, &cfg.SSH.Password, &cfg.SSH.Port, &cfg.SSH.PrivateKeyPath, &cfg.SSH.UpdateInterval,
			&cfg.SSH.Timeout, &cfg.SSH.DemoPathFmt,
			&cfg.Exports.BDEnabled, &cfg.Exports.ValveEnabled, &cfg.Exports.AuthorizedKeys)
	if err != nil {
		return cfg, c.db.DBErr(err)
	}

	return cfg, nil
}

func (c *configRepository) Init(ctx context.Context) error {
	if _, errRead := c.Read(ctx); errRead != nil {
		if errors.Is(errRead, domain.ErrNoResult) {
			// Insert a value so that the database will populate a row of defaults.
			if err := c.db.ExecInsertBuilder(ctx, c.db.Builder().
				Insert("config").
				SetMap(map[string]interface{}{
					"general_site_name": "New gbans site",
				})); err != nil {
				return err
			}

			return nil
		}

		return errRead
	}

	return nil
}

func (c *configRepository) Write(ctx context.Context, config domain.Config) error {
	return c.db.DBErr(c.db.ExecUpdateBuilder(ctx, c.db.Builder().
		Update("config").
		SetMap(map[string]interface{}{
			"general_site_name":                   config.General.SiteName,
			"general_steam_key":                   config.General.SteamKey,
			"general_mode":                        config.General.Mode,
			"general_file_serve_mode":             config.General.FileServeMode,
			"general_srcds_log_addr":              config.General.SrcdsLogAddr,
			"general_asset_url":                   config.General.AssetURL,
			"general_default_route":               config.General.DefaultRoute,
			"general_news_enabled":                config.General.NewsEnabled,
			"general_forums_enabled":              config.General.ForumsEnabled,
			"general_contests_enabled":            config.General.ContestsEnabled,
			"general_wiki_enabled":                config.General.WikiEnabled,
			"general_stats_enabled":               config.General.StatsEnabled,
			"general_servers_enabled":             config.General.ServersEnabled,
			"general_reports_enabled":             config.General.ReportsEnabled,
			"general_chatlogs_enabled":            config.General.ChatlogsEnabled,
			"general_demos_enabled":               config.General.DemosEnabled,
			"filters_enabled":                     config.Filters.Enabled,
			"filters_dry":                         config.Filters.Dry,
			"filters_ping_discord":                config.Filters.PingDiscord,
			"filters_max_weight":                  config.Filters.MaxWeight,
			"filters_warning_timeout":             config.Filters.WarningTimeout,
			"filters_check_timeout":               config.Filters.CheckTimeout,
			"filters_match_timeout":               config.Filters.MatchTimeout,
			"demo_cleanup_enabled":                config.Demo.DemoCleanupEnabled,
			"demo_cleanup_strategy":               config.Demo.DemoCleanupStrategy,
			"demo_cleanup_min_pct":                config.Demo.DemoCleanupMinPct,
			"demo_cleanup_mount":                  config.Demo.DemoCleanupMount,
			"demo_count_limit":                    config.Demo.DemoCountLimit,
			"patreon_enabled":                     config.Patreon.Enabled,
			"patreon_integrations_enabled":        config.Patreon.IntegrationsEnabled,
			"patreon_client_id":                   config.Patreon.ClientID,
			"patreon_client_secret":               config.Patreon.ClientSecret,
			"patreon_creator_access_token":        config.Patreon.CreatorAccessToken,
			"patreon_creator_refresh_token":       config.Patreon.CreatorRefreshToken,
			"discord_enabled":                     config.Discord.Enabled,
			"discord_bot_enabled":                 config.Discord.BotEnabled,
			"discord_integrations_enabled":        config.Discord.IntegrationsEnabled,
			"discord_app_id":                      config.Discord.AppID,
			"discord_app_secret":                  config.Discord.AppSecret,
			"discord_link_id":                     config.Discord.LinkID,
			"discord_token":                       config.Discord.Token,
			"discord_guild_id":                    config.Discord.GuildID,
			"discord_log_channel_id":              config.Discord.LogChannelID,
			"discord_public_log_channel_enabled":  config.Discord.PublicLogChannelEnable,
			"discord_public_log_channel_id":       config.Discord.PublicLogChannelID,
			"discord_public_match_log_channel_id": config.Discord.PublicMatchLogChannelID,
			"discord_mod_ping_role_id":            config.Discord.ModPingRoleID,
			"discord_vote_log_channel_id":         config.Discord.VoteLogChannelID,
			"discord_appeal_log_channel_id":       config.Discord.AppealLogChannelID,
			"discord_ban_log_channel_id":          config.Discord.BanLogChannelID,
			"discord_forum_log_channel_id":        config.Discord.ForumLogChannelID,
			"discord_word_filter_log_channel_id":  config.Discord.WordFilterLogChannelID,
			"discord_unregister_on_start":         config.Discord.UnregisterOnStart,
			"logging_level":                       config.Log.Level,
			"logging_file":                        config.Log.File,
			"sentry_sentry_dsn":                   config.Sentry.SentryDSN,
			"sentry_sentry_dsn_web":               config.Sentry.SentryDSNWeb,
			"sentry_sentry_trace":                 config.Sentry.SentryTrace,
			"sentry_sentry_sample_rate":           config.Sentry.SentrySampleRate,
			"ip2location_enabled":                 config.GeoLocation.Enabled,
			"ip2location_cache_path":              config.GeoLocation.CachePath,
			"ip2location_token":                   config.GeoLocation.Token,
			"debug_skip_open_id_validation":       config.Debug.SkipOpenIDValidation,
			"debug_add_rcon_log_address":          config.Debug.AddRCONLogAddress,
			"local_store_path_root":               config.LocalStore.PathRoot,
			"ssh_enabled":                         config.SSH.Enabled,
			"ssh_username":                        config.SSH.Username,
			"ssh_password":                        config.SSH.Password,
			"ssh_port":                            config.SSH.Port,
			"ssh_private_key_path":                config.SSH.PrivateKeyPath,
			"ssh_update_interval":                 config.SSH.UpdateInterval,
			"ssh_timeout":                         config.SSH.Timeout,
			"ssh_demo_path_fmt":                   config.SSH.DemoPathFmt,
			"exports_bd_enabled":                  config.Exports.BDEnabled,
			"exports_valve_enabled":               config.Exports.ValveEnabled,
			"exports_authorized_keys":             config.Exports.AuthorizedKeys,
		})))
}
