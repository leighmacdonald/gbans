import { z } from 'zod/v4';
import { schemaQueryFilter } from './query.ts';

export const PermissionLevel = {
    Banned: 0,
    Guest: 1,
    User: 10,
    Reserved: 15,
    Editor: 25,
    Moderator: 50,
    Admin: 100
} as const;

export const PermissionLevelEnum = z.enum(PermissionLevel);
export type PermissionLevelEnum = z.infer<typeof PermissionLevelEnum>;

export const PermissionLevelCollection = [
    PermissionLevel.Banned,
    PermissionLevel.Guest,
    PermissionLevel.User,
    PermissionLevel.Reserved,
    PermissionLevel.Editor,
    PermissionLevel.Moderator,
    PermissionLevel.Admin
];

export const permissionLevelString = (level: PermissionLevelEnum) => {
    switch (level) {
        case PermissionLevel.Admin:
            return 'Admin';
        case PermissionLevel.Editor:
            return 'Editor';
        case PermissionLevel.Banned:
            return 'Banned';
        case PermissionLevel.User:
            return 'User';
        case PermissionLevel.Moderator:
            return 'Moderator';
        case PermissionLevel.Reserved:
            return 'VIP';
        case PermissionLevel.Guest:
            return 'Guest';
        default:
            return 'Unknown';
    }
};

export const profileState = {
    Incomplete: 0,
    Setup: 1
} as const;

export const profileStateEnum = z.enum(profileState);
export type profileStateEnum = z.infer<typeof profileStateEnum>;

export const communityVisibilityState = {
    Private: 1,
    FriendOnly: 2,
    Public: 3
} as const;

export const communityVisibilityStateEnum = z.enum(communityVisibilityState);
export type communityVisibilityStateEnum = z.infer<typeof communityVisibilityStateEnum>;

export const NotificationSeverity = {
    SeverityInfo: 0,
    SeverityWarn: 1,
    SeverityError: 2
} as const;

export const NotificationSeverityEnum = z.enum(NotificationSeverity);
export type NotificationSeverityEnum = z.infer<typeof NotificationSeverityEnum>;

export const schemaUserProfile = z.object({
    steam_id: z.string(),
    permission_level: PermissionLevelEnum,
    discord_id: z.string(),
    patreon_id: z.string(),
    name: z.string(),
    avatar_hash: z.string(),
    ban_id: z.number(),
    muted: z.boolean(),
    created_on: z.date(),
    updated_on: z.date(),
    playerqueue_chat_status: z.enum(['readwrite', 'readonly', 'noaccess']).default('readwrite')
    // playerqueue_chat_reason: z.string()
});

export type UserProfile = z.infer<typeof schemaUserProfile>;

export const schemaUserNotification = z.object({
    person_notification_id: z.number(),
    steam_id: z.string(),
    read: z.boolean(),
    deleted: z.boolean(),
    severity: NotificationSeverityEnum,
    message: z.string(),
    link: z.string(),
    count: z.number(),
    author: schemaUserProfile.optional(),
    created_on: z.date()
});

export type UserNotification = z.infer<typeof schemaUserNotification>;

export const schemaPerson = z.object({
    steam_id: z.string(),
    permission_level: PermissionLevelEnum,
    discord_id: z.string(),
    patreon_id: z.string(),
    name: z.string(),
    avatar_hash: z.string(),
    ban_id: z.number(),
    muted: z.boolean(),
    created_on: z.date(),
    updated_on: z.date(),
    playerqueue_chat_status: z.enum(['readwrite', 'readonly', 'noaccess']).default('readwrite'),
    // PlayerSummaries shape
    community_visibility_state: communityVisibilityStateEnum,
    profile_state: profileStateEnum,
    persona_name: z.string(),
    profile_url: z.string(),
    avatar_medium: z.string(),
    persona_state: z.number(),
    realname: z.string(),
    primary_clan_id: z.string(), // ? should be number
    time_created: z.number(),
    persona_state_flags: z.number(),
    loc_country_code: z.string(),
    loc_state_code: z.string(),
    loc_city_id: z.number(),

    // BanStates
    community_banned: z.boolean(),
    vac_bans: z.number(),
    game_bans: z.number(),
    economy_ban: z.string(),
    days_since_last_ban: z.number(),
    updated_on_steam: z.date(),
    ip_addr: z.string()
});

export type Person = z.infer<typeof schemaPerson>;

export const schemaSteamValidate = z.object({
    steam_id: z.string(),
    hash: z.string(),
    personaname: z.string()
});

export type SteamValidate = z.infer<typeof schemaSteamValidate>;

export const schemaPersonSettings = z.object({
    person_settings_id: z.number(),
    steam_id: z.string(),
    forum_signature: z.string(),
    forum_profile_messages: z.boolean(),
    stats_hidden: z.boolean(),
    center_projectiles: z.boolean(),
    created_on: z.date(),
    updated_on: z.date()
});

export type PersonSettings = z.infer<typeof schemaPersonSettings>;

export const schemaPlayerProfile = z.object({
    player: schemaPerson,
    friends: z.array(schemaPerson),
    settings: schemaPersonSettings
});

export type PlayerProfile = z.infer<typeof schemaPlayerProfile>;

export const schemaPlayerQuery = z
    .object({
        target_id: z.string(),
        personaname: z.string(),
        ip: z.string(),
        staff_only: z.boolean()
    })
    .merge(schemaQueryFilter);

export type PlayerQuery = z.infer<typeof schemaPlayerQuery>;

export const schemaPersonIPRecord = z.object({
    ip_addr: z.string(),
    created_on: z.date(),
    city_name: z.string(),
    country_name: z.string(),
    country_code: z.string(),
    as_name: z.string(),
    as_num: z.number(),
    isp: z.string(),
    usage_type: z.string(),
    threat: z.string(),
    domain: z.string()
});

export type PersonIPRecord = z.infer<typeof schemaPersonIPRecord>;

export const schemaPersonConnection = z.object({
    person_connection_id: z.bigint(),
    ip_addr: z.string(),
    steam_id: z.string(),
    persona_name: z.string(),
    created_on: z.date(),
    ip_info: schemaPersonIPRecord,
    server_id: z.number().optional(),
    server_name_short: z.string().optional(),
    server_name: z.string().optional()
});

export type PersonConnection = z.infer<typeof schemaPersonConnection>;

export const schemaPersonMessage = z.object({
    person_message_id: z.number(),
    steam_id: z.string(),
    persona_name: z.string(),
    server_name: z.string(),
    server_id: z.number(),
    body: z.string(),
    team: z.boolean(),
    created_on: z.date(),
    auto_filter_flagged: z.number(),
    avatar_hash: z.string(),
    pattern: z.string()
});

export type PersonMessage = z.infer<typeof schemaPersonMessage>;

export const schemaMessageQuery = z
    .object({
        personaname: z.string().optional(),
        source_id: z.string().optional(),
        query: z.string().optional(),
        server_id: z.number().optional(),
        date_start: z.date().optional(),
        date_end: z.date().optional(),
        match_id: z.string().optional(),
        auto_filter_flagged: z.boolean().optional()
    })
    .merge(schemaQueryFilter);

export type MessageQuery = z.infer<typeof schemaMessageQuery>;

export const schemaConnectionQuery = schemaQueryFilter.extend({
    cidr: z.cidrv4().optional(),
    source_id: z.string().optional(),
    server_id: z.number().optional(),
    asn: z.number().optional(),
    network: z.string().optional(),
    sid64: z.string().optional()
});

export type ConnectionQuery = z.infer<typeof schemaConnectionQuery>;

export const schemaIPQuery = z.object({
    ip: z.ipv4()
});
export type IPQuery = z.infer<typeof schemaIPQuery>;

export const schemaNetworkLocation = z.object({
    cidr: z.string(),
    country_code: z.string(),
    country_name: z.string(),
    region_name: z.string(),
    city_name: z.string(),
    lat_long: z.object({
        latitude: z.number(),
        longitude: z.number()
    })
});

export type NetworkLocation = z.infer<typeof schemaNetworkLocation>;

export const schemaNetworkASN = z.object({
    cidr: z.string(),
    as_num: z.number(),
    as_name: z.string()
});

export const schemaNetworkProxy = z.object({
    cidr: z.string(),
    proxy_type: z.string(),
    country_code: z.string(),
    country_name: z.string(),
    region_name: z.string(),
    city_name: z.string(),
    isp: z.string(),
    domain: z.string(),
    usage_type: z.string(),
    asn: z.number(),
    as: z.string(),
    last_seen: z.string(),
    threat: z.string()
});

export const schemaNetworkDetails = z.object({
    location: schemaNetworkLocation,
    asn: schemaNetworkASN,
    proxy: schemaNetworkProxy
});

export type NetworkDetails = z.infer<typeof schemaNetworkDetails>;

export const schemaPermissionUpdate = z.object({
    permission_level: PermissionLevelEnum
});
export type PermissionUpdate = z.infer<typeof schemaPermissionUpdate>;

export const schemaDiscordUser = z.object({
    username: z.string(),
    id: z.string(),
    avatar: z.string(),
    mfa_enabled: z.boolean(),
    created_on: z.date(),
    updated_on: z.date()
});

export type DiscordUser = z.infer<typeof schemaDiscordUser>;
