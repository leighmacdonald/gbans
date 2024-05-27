import { apiCall } from './common.ts';

export const apiSaveSettings = async (settings: Config) => {
    return await apiCall(`/api/settings`, 'PUT', settings);
};

export const apiGetSettings = (): Config => {
    return {
        debug: undefined,
        demos: undefined,
        discord: undefined,
        exports: undefined,
        filters: undefined,
        general: {
            srcds_log_addr: '',
            file_serve_mode: 'local',
            steam_key: '12345',
            mode: 'release',
            site_name: 'Site name'
        },
        ip2location: undefined,
        local_store: undefined,
        logging: undefined,
        patreon: undefined,
        sentry: undefined,
        ssh: undefined
    };
};

export type Config = {
    general: General;
    filters: Filters;
    demos: Demos;
    patreon: Patreon;
    discord: Discord;
    logging: Logging;
    sentry: Sentry;
    ip2location: IP2Location;
    debug: Debug;
    local_store: LocalStore;
    ssh: SSH;
    exports: Exports;
};

type General = {
    site_name: string;
    steam_key: string;
    mode: 'release' | 'debug' | 'test';
    file_serve_mode: 'local';
    srcds_log_addr: string;
};

type Filters = {
    enabled: boolean;
    dry: boolean;
    ping_discord: boolean;
    max_weight: number;
    warning_timeout: number;
    warning_limit: number;
    check_timeout: number;
    match_timeout: number;
};

type Demos = {
    demo_cleanup_enabled: boolean;
    demo_cleanup_strategy: 'pctfree' | 'count';
    demo_cleanup_min_pct: number;
    demo_cleanup_mount: string;
    demo_count_limit: number;
};

type Patreon = {
    enabled: boolean;
    client_id: string;
    client_secret: string;
    creator_access_token: string;
    creator_refresh_token: string;
};

type Discord = {
    enabled: boolean;
    app_id: string;
    app_secret: string;
    link_id: string;
    token: string;
    guild_id: string;
    log_channel_id: string;
    public_log_channel_enable: boolean;
    public_log_channel_id: string;
    public_match_log_channel_id: string;
    mod_ping_role_id: string;
    unregister_on_start: boolean;
};

type Logging = {
    level: 'debug' | 'info' | 'warn' | 'error';
    file: string;
    report_caller: boolean;
    full_timestamp: boolean;
};

type Sentry = {
    sentry_dsn: string;
    sentry_dsn_web: string;
    sentry_trace: boolean;
    sentry_sample_rate: number;
};

type IP2Location = {
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
    port: number;
    private_key_path: string;
    password: string;
    update_interval: number;
    timeout: number;
    demo_path_fmt: string;
};

type Exports = {
    bd_enabled: boolean;
    valve_enabled: boolean;
    authorized_keys: string[];
};
