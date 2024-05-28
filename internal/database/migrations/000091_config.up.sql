BEGIN;

CREATE DOMAIN uint2 AS int4
    CHECK(VALUE >= 0 AND VALUE < 65536);

CREATE TABLE config
(
    general_site_name text not null default 'gbans' CHECK ( length(general_site_name) > 0 ),
    general_steam_key text not null default '' CHECK ( length(general_steam_key) IN (0, 32) ),
    general_mode text not null default 'release' CHECK ( general_mode IN ('release' , 'debug' , 'test') ),
    general_file_serve_mode text not null default 'local' CHECK ( general_file_serve_mode IN ('local') ),
    general_srcds_log_addr text not null default ':27115',

    filters_enabled bool not null default false,
    filters_dry bool not null default false,
    filters_ping_discord bool not null default true,
    filters_max_weight int not null default 10,
    filters_warning_timeout int not null default 6220800,
    filters_check_timeout int not null default 5,
    filters_match_timeout int not null default 7200,

    demo_cleanup_enabled bool NOT NULL default false,
    demo_cleanup_strategy text NOT NULL default 'pctfree' CHECK ( demo_cleanup_strategy IN ('pctfree', 'count') ),
    demo_cleanup_min_pct int NOT NULL default 80,
    demo_cleanup_mount text NOT NULL default '/',
    demo_count_limit int NOT NULL default 1000,

    patreon_enabled bool NOT NULL default false,
    patreon_client_id text NOT NULL default '',
    patreon_client_secret text not null default '',
    patreon_creator_access_token text not null default '',
    patreon_creator_refresh_token text not null default '',

    discord_enabled bool not null default false,
    discord_app_id text not null default '',
    discord_app_secret text not null default '',
    discord_link_id text not null default '',
    discord_token text not null default '',
    discord_guild_id text not null default '',
    discord_log_channel_id text not null default '',
    discord_public_log_channel_enabled bool not null default false,
    discord_public_log_channel_id text not null default '',
    discord_public_match_log_channel_id text not null default '',
    discord_mod_ping_role_id text not null default '',
    discord_unregister_on_start bool not null default false,

    logging_level text not null default 'error' CHECK ( logging_level in ('debug', 'info', 'warn', 'error') ),
    logging_file text not null default '',

    sentry_sentry_dsn text not null default '',
    sentry_sentry_dsn_web text not null default '',
    sentry_sentry_trace bool not null default false,
    sentry_sentry_sample_rate float not null default 0.1,

    ip2location_enabled bool not null default false,
    ip2location_cache_path text not null default '.cache',
    ip2location_token text not null default '',

    debug_skip_open_id_validation bool not null default false,
    debug_add_rcon_log_address text not null default '127.0.0.1:27115',

    local_store_path_root text not null default './assets',

    ssh_enabled bool not null default false,
    ssh_username text not null default '',
    ssh_password text not null default '',
    ssh_port uint2 not null  default 22,
    ssh_private_key_path text not null default '',
    ssh_update_interval int not null default 60,
    ssh_timeout int not null default 10,
    ssh_demo_path_fmt text not null default '~/srcds-%s/tf/stv_demos/complete/',

    exports_bd_enabled bool not null default false,
    exports_valve_enabled bool not null default false,
    exports_authorized_keys text[] not null default '{}'
);

COMMIT;
