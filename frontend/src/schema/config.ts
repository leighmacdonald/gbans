import { z } from 'zod/v4';
import { Action } from './asset.ts';

const coercedNumber = z.string().transform(Number);

export const schemaAnticheat = z.object({
    enabled: z.boolean(),
    action: Action,
    duration: z.coerce.number().int(),
    max_aim_snap: z.coerce.number().int(),
    max_psilent: z.coerce.number().int(),
    max_bhop: z.coerce.number().int(),
    max_fake_ang: z.coerce.number().int(),
    max_cmd_num: z.coerce.number().int(),
    max_too_many_connections: z.coerce.number().int(),
    max_cheat_cvar: z.coerce.number().int(),
    max_oob_var: z.coerce.number().int(),
    max_invalid_user_cmd: z.coerce.number().int()
});
export const schemaSentry = z.object({
    sentry_dsn: z.string(),
    sentry_dsn_web: z.string(),
    sentry_trace: z.boolean(),
    sentry_sample_rate: z.number()
});

export const schemaGeneral = z.object({
    site_name: z.string().min(1).max(32),
    file_serve_mode: z.enum(['local']),
    mode: z.enum(['release', 'debug', 'test']),
    srcds_log_addr: z.string(),
    default_route: z.string(),
    asset_url: z.string(),
    contests_enabled: z.boolean(),
    news_enabled: z.boolean(),
    forums_enabled: z.boolean(),
    wiki_enabled: z.boolean(),
    stats_enabled: z.boolean(),
    servers_enabled: z.boolean(),
    reports_enabled: z.boolean(),
    chatlogs_enabled: z.boolean(),
    demos_enabled: z.boolean(),
    speedruns_enabled: z.boolean(),
    playerqueue_enabled: z.boolean()
});

export const schemaFilters = z.object({
    enabled: z.boolean(),
    warning_timeout: z.number().min(1).max(1000000),
    warning_limit: z.number().min(0).max(1000),
    dry: z.boolean(),
    ping_discord: z.boolean(),
    max_weight: z.number().min(1).max(1000),
    check_timeout: z.number().min(5).max(300),
    match_timeout: z.number().min(1).max(1000)
});

export const schemaDemos = z.object({
    demo_cleanup_enabled: z.boolean(),
    demo_cleanup_strategy: z.enum(['pctfree', 'count']),
    demo_cleanup_min_pct: z.number().min(0).max(100),
    demo_cleanup_mount: z.string().startsWith('/'), // windows?
    demo_count_limit: z.number(),
    demo_parser_url: z.string()
});

export const schemaPatreon = z.object({
    enabled: z.boolean(),
    integrations_enabled: z.boolean(),
    client_id: z.string(),
    client_secret: z.string(),
    creator_access_token: z.string(),
    creator_refresh_token: z.string()
});

export const schemaDiscord = z
    .object({
        enabled: z.boolean(),
        bot_enabled: z.boolean(),
        integrations_enabled: z.boolean(),
        app_id: z.string().refine((arg) => arg.length == 0 || arg.length == 18),
        app_secret: z.string(),
        link_id: z.string(),
        token: z.string(),
        guild_id: z.string(),
        log_channel_id: z.string(),
        public_log_channel_enable: z.boolean(),
        public_log_channel_id: z.string(),
        public_match_log_channel_id: z.string(),
        mod_ping_role_id: z.string(),
        vote_log_channel_id: z.string(),
        appeal_log_channel_id: z.string(),
        ban_log_channel_id: z.string(),
        forum_log_channel_id: z.string(),
        word_filter_log_channel_id: z.string().optional(),
        kick_log_channel_id: z.string(),
        playerqueue_channel_id: z.string(),
        anticheat_channel_id: z.string(),
        seed_channel_id: z.string()
    })
    .refine((data) => {
        if (!data.bot_enabled) {
            return true;
        }
        if (data.log_channel_id == '') {
            return false;
        }

        return true;
    });

export const schemaLogging = z.object({
    level: z.enum(['debug', 'info', 'warn', 'error']),
    file: z.string(),
    http_enabled: z.boolean(),
    http_otel_enabled: z.boolean(),
    http_level: z.enum(['debug', 'info', 'warn', 'error'])
});

export const schemaGeo = z.object({
    enabled: z.boolean(),
    cache_path: z.string(),
    token: z.string().refine((arg) => arg.length == 0 || arg.length == 64)
});

export const schemaDebug = z.object({
    skip_open_id_validation: z.boolean(),
    add_rcon_log_address: z.string()
});

export const schemaLocalStore = z.object({
    path_root: z.string()
});

export const schemaSSH = z.object({
    enabled: z.boolean(),
    username: z.string(),
    port: coercedNumber.pipe(z.number().min(1).max(65535)),
    private_key_path: z.string(),
    password: z.string(),
    update_interval: coercedNumber.pipe(z.number().positive()),
    timeout: coercedNumber.pipe(z.number().positive()),
    demo_path_fmt: z.string(),
    stac_path_fmt: z.string()
});

export const schemaExports = z.object({
    bd_enabled: z.boolean(),
    valve_enabled: z.boolean(),
    authorized_keys: z.string()
});

export const schemaNetwork = z.object({
    sdr_enabled: z.boolean(),
    sdr_dns_enabled: z.boolean(),
    cf_key: z.string(),
    cf_email: z.string(),
    cf_zone_id: z.string()
});

export const schemaConfig = z.object({
    general: schemaGeneral,
    filters: schemaFilters,
    demo: schemaDemos,
    patreon: schemaPatreon,
    discord: schemaDiscord,
    network: schemaNetwork,
    log: schemaLogging,
    sentry: schemaSentry,
    geo_location: schemaGeo,
    debug: schemaDebug,
    local_store: schemaLocalStore,
    ssh: schemaSSH,
    exports: schemaExports,
    anticheat: schemaAnticheat
});

export type Config = z.infer<typeof schemaConfig>;
