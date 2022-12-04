import { apiCall, PermissionLevel, QueryFilter, TimeStamped } from './common';
import SteamID from 'steamid';
import { parseDateTime } from '../util/text';

export enum profileState {
    Incomplete = 0,
    Setup = 1
}

export enum communityVisibilityState {
    Private = 1,
    FriendOnly = 2,
    Public = 3
}

export interface UserProfile extends TimeStamped {
    steam_id: SteamID;
    permission_level: PermissionLevel;
    discord_id: string;
    name: string;
    avatar: string;
    avatarfull: string;
    ban_id: number;
    muted: boolean;
}

export interface Person extends UserProfile {
    // PlayerSummaries shape
    steamid: SteamID;
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
}

const validSteamIdKeys = ['target_id', 'source_id', 'steam_id', 'author_id'];

export const applySteamId = (key: string, value: unknown) => {
    if (validSteamIdKeys.includes(key)) {
        try {
            return new SteamID(`${value}`);
        } catch (e) {
            return new SteamID('');
        }
    }
    return value;
};

export const apiGetProfile = async (query: string) =>
    await apiCall<PlayerProfile>(`/api/profile?query=${query}`, 'GET');

export const apiGetCurrentProfile = async () =>
    await apiCall<UserProfile>(`/api/current_profile`, 'GET');

export const apiGetPeople = async () =>
    await apiCall<Person[]>(`/api/players`, 'GET');

export const apiLinkDiscord = async (opts: { code: string }) =>
    await apiCall(`/api/auth/discord?code=${opts.code}`, 'GET');

export interface FindProfileProps {
    query: string;
}

export const apiGetResolveProfile = async (opts: FindProfileProps) =>
    await apiCall<Person, FindProfileProps>(
        '/api/resolve_profile',
        'POST',
        opts
    );

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
    connection_id: bigint;
    ip_addr: string;
    steam_id: SteamID;
    persona_name: string;
    created_on: Date;
    ip_info: PersonIPRecord;
}

export interface PersonMessage {
    person_message_id: number;
    steam_id: SteamID;
    persona_name: string;
    server_name: string;
    server_id: number;
    body: string;
    team: boolean;
    created_on: Date;
}

export const apiGetPersonConnections = async (steam_id: SteamID) =>
    await apiCall<PersonConnection[]>(`/api/connections/${steam_id}`, 'GET');

export const apiGetPersonMessages = async (steam_id: SteamID) => {
    const resp = await apiCall<PersonMessage[]>(
        `/api/messages/${steam_id}`,
        'GET'
    );
    resp.result = resp.result?.map((msg) => {
        return {
            ...msg,
            created_on: parseDateTime(msg.created_on as unknown as string)
        };
    });
    return resp;
};

export const apiGetMessageContext = async (message_id: number) => {
    const resp = await apiCall<PersonMessage[]>(
        `/api/message/${message_id}/context`,
        'GET'
    );
    resp.result = resp.result?.map((msg) => {
        return {
            ...msg,
            created_on: parseDateTime(msg.created_on as unknown as string)
        };
    });
    return resp;
};

export interface MessageQuery extends QueryFilter<PersonMessage> {
    persona_name?: string;
    steam_id?: string;
    query?: string;
    server_id?: number;
    sent_after?: Date;
    sent_before?: Date;
}

export const apiGetMessages = async (opts: MessageQuery) => {
    const resp = await apiCall<PersonMessage[]>(`/api/messages`, 'POST', opts);
    resp.result = resp.result?.map((msg) => {
        return {
            ...msg,
            created_on: parseDateTime(msg.created_on as unknown as string)
        };
    });
    return resp;
};
