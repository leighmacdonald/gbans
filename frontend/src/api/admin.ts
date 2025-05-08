import { apiCall } from './common.ts';

export const apiSaveSettings = async (settings: Config) => {
    return await apiCall(`/api/config`, 'PUT', settings);
};

export const apiGetSettings = async () => {
    return await apiCall<Config>('/api/config', 'GET');
};

export type Config = {
    general: General;
    filters: Filters;
    demo: Demos;
    patreon: Patreon;
    discord: Discord;
    network: Network;
    log: Logging;
    sentry: Sentry;
    geo_location: GeoLocation;
    debug: Debug;
    local_store: LocalStore;
    ssh: SSH;
    exports: Exports;
    anticheat: Anticheat;
};

type Network = {
    sdr_enabled: boolean;
    sdr_dns_enabled: boolean;
    cf_key: string;
    cf_email: string;
};

export enum Action {
    Ban = 'ban',
    Kick = 'kick',
    Gag = 'gag'
}

export const ActionColl = [Action.Ban, Action.Kick, Action.Gag];

type Anticheat = {
    enabled: boolean;
    action: Action;
    duration: number;
    max_aim_snap: number;
    max_psilent: number;
    max_bhop: number;
    max_fake_ang: number;
    max_cmd_num: number;
    max_too_many_connections: number;
    max_cheat_cvar: number;
    max_oob_var: number;
    max_invalid_user_cmd: number;
};

type General = {
    site_name: string;
    mode: 'release' | 'debug' | 'test';
    file_serve_mode: 'local';
    srcds_log_addr: string;
    asset_url: string;
    default_route: string;
    news_enabled: boolean;
    forums_enabled: boolean;
    contests_enabled: boolean;
    wiki_enabled: boolean;
    stats_enabled: boolean;
    servers_enabled: boolean;
    reports_enabled: boolean;
    chatlogs_enabled: boolean;
    demos_enabled: boolean;
    speedruns_enabled: boolean;
    playerqueue_enabled: boolean;
};

type Filters = {
    enabled: boolean;
    dry: boolean;
    ping_discord: boolean;
    max_weight: string;
    warning_timeout: string;
    warning_limit: string;
    check_timeout: string;
    match_timeout: string;
};

type Demos = {
    demo_cleanup_enabled: boolean;
    demo_cleanup_strategy: 'pctfree' | 'count';
    demo_cleanup_min_pct: string;
    demo_cleanup_mount: string;
    demo_count_limit: string;
    demo_parser_url: string;
};

type Patreon = {
    enabled: boolean;
    integrations_enabled: boolean;
    client_id: string;
    client_secret: string;
    creator_access_token: string;
    creator_refresh_token: string;
};

type Discord = {
    enabled: boolean;
    bot_enabled: boolean;
    integrations_enabled: boolean;
    app_id: string;
    app_secret: string;
    link_id: string;
    token: string;
    guild_id: string;
    anticheat_channel_id: string;
    log_channel_id: string;
    public_log_channel_enable: boolean;
    public_log_channel_id: string;
    public_match_log_channel_id: string;
    mod_ping_role_id: string;
    vote_log_channel_id: string;
    appeal_log_channel_id: string;
    ban_log_channel_id: string;
    forum_log_channel_id: string;
    kick_log_channel_id: string;
    word_filter_log_channel_id: string;
    playerqueue_channel_id: string;
};

type LogLevels = 'debug' | 'info' | 'warn' | 'error';

type Logging = {
    level: LogLevels;
    file: string;
    http_enabled: boolean;
    http_otel_enabled: boolean;
    http_level: LogLevels;
};

type Sentry = {
    sentry_dsn: string;
    sentry_dsn_web: string;
    sentry_trace: boolean;
    sentry_sample_rate: string;
};

type GeoLocation = {
    enabled: boolean;
    cache_path: string;
    token: string;
};

type Debug = {
    skip_open_id_validation: boolean;
    add_rcon_log_address: string;
};

type LocalStore = {
    path_root: string;
};

type SSH = {
    enabled: boolean;
    username: string;
    port: string;
    private_key_path: string;
    password: string;
    update_interval: string;
    timeout: string;
    demo_path_fmt: string;
    stac_path_fmt: string;
};

type Exports = {
    bd_enabled: boolean;
    valve_enabled: boolean;
    authorized_keys: string;
};
