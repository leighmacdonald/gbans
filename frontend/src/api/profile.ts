import { LazyResult } from '../util/table.ts';
import { parseDateTime } from '../util/text.tsx';
import {
    apiCall,
    PermissionLevel,
    QueryFilter,
    TimeStamped,
    transformCreatedOnDate,
    transformTimeStampedDates
} from './common';

export const defaultAvatarHash = 'fef49e7fa7e1997310d705b2a6158ff8dc1cdfeb';

export enum profileState {
    Incomplete = 0,
    Setup = 1
}

export enum communityVisibilityState {
    Private = 1,
    FriendOnly = 2,
    Public = 3
}

export enum NotificationSeverity {
    SeverityInfo,
    SeverityWarn,
    SeverityError
}

export interface UserNotification {
    person_notification_id: number;
    steam_id: string;
    read: boolean;
    deleted: boolean;
    severity: NotificationSeverity;
    message: string;
    link: string;
    count: number;
    author?: UserProfile;
    created_on: Date;
}

export interface UserProfile extends TimeStamped {
    steam_id: string;
    permission_level: PermissionLevel;
    discord_id: string;
    patreon_id: string;
    name: string;
    avatarhash: string;
    ban_id: number;
    muted: boolean;
}

export interface Person extends UserProfile {
    // PlayerSummaries shape
    steamid: string;
    communityvisibilitystate: communityVisibilityState;
    profilestate: profileState;
    personaname: string;
    profileurl: string;
    avatarmedium: string;
    avatarhash: string;
    personastate: number;
    realname: string;
    primaryclanid: string; // ? should be number
    timecreated: number;
    personastateflags: number;
    loccountrycode: string;
    locstatecode: string;
    loccityid: number;

    // BanStates
    community_banned: boolean;
    vac_bans: number;
    game_bans: number;
    economy_ban: string;
    days_since_last_ban: number;
    updated_on_steam: Date;
    ip_addr: string;
}

export interface PlayerProfile {
    player: Person;
    friends?: Person[];
    settings: PersonSettings;
}

export const apiGetProfile = async (query: string, abortController?: AbortController) => {
    const profile = await apiCall<PlayerProfile>(`/api/profile?query=${query}`, 'GET', undefined, abortController);
    profile.player = transformTimeStampedDates(profile.player);
    return profile;
};

export const apiGetCurrentProfile = async () => apiCall<UserProfile>(`/api/current_profile`, 'GET', undefined);

export interface PlayerQuery extends QueryFilter {
    target_id: string;
    personaname: string;
    ip: string;
    staff_only: boolean;
}

export const apiSearchPeople = async (opts: PlayerQuery, abortController?: AbortController) => {
    const people = await apiCall<LazyResult<Person>>(`/api/players`, 'POST', opts, abortController);
    people.data = people.data.map(transformTimeStampedDates);
    return people;
};

export interface PersonIPRecord {
    ip_addr: string;
    created_on: Date;
    city_name: string;
    country_name: string;
    country_code: string;
    as_name: string;
    as_num: number;
    isp: string;
    usage_type: string;
    threat: string;
    domain: string;
}

export interface PersonConnection {
    person_connection_id: bigint;
    ip_addr: string;
    steam_id: string;
    persona_name: string;
    created_on: Date;
    ip_info: PersonIPRecord;
    server_id?: number;
    server_name_short?: string;
    server_name?: string;
}

export interface PersonMessage {
    person_message_id: number;
    steam_id: string;
    persona_name: string;
    server_name: string;
    server_id: number;
    body: string;
    team: boolean;
    created_on: Date;
    auto_filter_flagged: number;
    avatar_hash: string;
    pattern: string;
}

export const apiGetMessageContext = async (messageId: number, padding: number = 5) => {
    const resp = await apiCall<PersonMessage[]>(`/api/message/${messageId}/context/${padding}`, 'GET');
    return resp.map((msg) => {
        return {
            ...msg,
            created_on: parseDateTime(msg.created_on as unknown as string)
        };
    });
};

export interface MessageQuery extends QueryFilter {
    personaname?: string;
    source_id?: string;
    query?: string;
    server_id?: number;
    date_start?: Date;
    date_end?: Date;
    match_id?: string;
    auto_filter_flagged?: boolean;
}

export const apiGetMessages = async (opts: MessageQuery, abortController?: AbortController) => {
    const resp = await apiCall<PersonMessage[], MessageQuery>(
        `/api/messages`,
        'POST',
        {
            ...opts,
            date_start: (opts.date_start ?? '') == '' ? undefined : opts.date_start,
            date_end: (opts.date_end ?? '') == '' ? undefined : opts.date_end
        },
        abortController
    );

    return resp.map((msg) => {
        return {
            ...msg,
            created_on: parseDateTime(msg.created_on as unknown as string)
        };
    });
};

export const apiGetNotifications = async (abortController?: AbortController) => {
    const resp = await apiCall<UserNotification[]>(`/api/notifications`, 'GET', undefined, abortController);
    return resp.map((msg) => {
        return { ...msg, created_on: parseDateTime(msg.created_on as unknown as string) };
    });
};

export const apiNotificationsMarkAllRead = async () => {
    return await apiCall(`/api/notifications/all`, 'POST', undefined);
};

export const apiNotificationsMarkRead = async (message_ids: number[]) => {
    return await apiCall(`/api/notifications`, 'POST', { message_ids });
};

export const apiNotificationsDeleteAll = async () => {
    return await apiCall(`/api/notifications/all`, 'DELETE', undefined);
};

export const apiNotificationsDelete = async (message_ids: number[]) => {
    return await apiCall(`/api/notifications`, 'DELETE', { message_ids });
};

export type ConnectionQuery = {
    cidr?: string;
    source_id?: string;
    server_id?: number;
    asn?: number;
} & QueryFilter;

export const apiGetConnections = async (opts: ConnectionQuery, abortController?: AbortController) => {
    const resp = await apiCall<LazyResult<PersonConnection>, ConnectionQuery>(
        `/api/connections`,
        'POST',
        opts,
        abortController
    );

    resp.data = resp.data.map(transformCreatedOnDate);
    return resp;
};

export type IPQuery = {
    ip: string;
};

export type NetworkLocation = {
    cidr: string;
    country_code: string;
    country_name: string;
    region_name: string;
    city_name: string;
    lat_long: {
        latitude: number;
        longitude: number;
    };
};

export type NetworkASN = {
    cidr: string;
    as_num: number;
    as_name: string;
};

export type NetworkProxy = {
    cidr: string;
    proxy_type: string;
    country_code: string;
    country_name: string;
    region_name: string;
    city_name: string;
    isp: string;
    domain: string;
    usage_type: string;
    asn: number;
    as: string;
    last_seen: string;
    threat: string;
};

export type NetworkDetails = {
    location: NetworkLocation;
    asn: NetworkASN;
    proxy: NetworkProxy;
};

export const apiGetNetworkDetails = async (opts: IPQuery, abortController?: AbortController) => {
    return await apiCall<NetworkDetails, IPQuery>(`/api/network`, 'POST', opts, abortController);
};

interface PermissionUpdate {
    permission_level: PermissionLevel;
}

export const apiUpdatePlayerPermission = async (
    steamId: string,
    args: PermissionUpdate,
    abortController?: AbortController
) =>
    transformTimeStampedDates(
        await apiCall<Person, PermissionUpdate>(`/api/player/${steamId}/permissions`, 'PUT', args, abortController)
    );

export interface PersonSettings extends TimeStamped {
    person_settings_id: number;
    steam_id: string;
    forum_signature: string;
    forum_profile_messages: boolean;
    stats_hidden: boolean;
    center_projectiles: boolean;
}

export const apiGetPersonSettings = async (abortController?: AbortController) => {
    return transformTimeStampedDates(
        await apiCall<PersonSettings>(`/api/current_profile/settings`, 'GET', undefined, abortController)
    );
};

export const apiSavePersonSettings = async (
    forum_signature: string,
    forum_profile_messages: boolean,
    stats_hidden: boolean,
    center_projectiles: boolean,
    abortController?: AbortController
) => {
    return transformTimeStampedDates(
        await apiCall<PersonSettings>(
            `/api/current_profile/settings`,
            'POST',
            { forum_signature, forum_profile_messages, stats_hidden, center_projectiles },
            abortController
        )
    );
};

export type DiscordUser = {
    username: string;
    id: string;
    avatar: string;
    mfa_enabled: boolean;
} & TimeStamped;

export const apiDiscordUser = async () => {
    return transformTimeStampedDates(await apiCall<DiscordUser>('/api/discord/user'));
};

export const discordAvatarURL = (user: DiscordUser) => {
    return `https://cdn.discordapp.com/avatars/${user.id}/${user.avatar}.png`;
};

export const apiDiscordLogout = async () => {
    return await apiCall<DiscordUser>('/api/discord/logout');
};
