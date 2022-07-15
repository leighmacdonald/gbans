import { apiCall, PermissionLevel, TimeStamped } from './common';
import { SteamID } from './const';

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
}

export interface Person extends TimeStamped {
    // PlayerSummaries shape
    steamid: SteamID;
    communityvisibilitystate: communityVisibilityState;
    profilestate: profileState;
    personaname: string;
    profileurl: string;
    avatar: string;
    avatarmedium: string;
    avatarfull: string;
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

    // Custom attributes
    steam_id: SteamID;
    permission_level: PermissionLevel;
    discord_id: string;
    ip_addr: string;
}

export interface PlayerProfile {
    player: Person;
    friends?: Person[];
}

export const apiGetProfile = async (query: SteamID): Promise<PlayerProfile> =>
    await apiCall<PlayerProfile>(`/api/profile?query=${query}`, 'GET');

export const apiGetCurrentProfile = async (): Promise<UserProfile> =>
    await apiCall<UserProfile>(`/api/current_profile`, 'GET');

export const apiGetPeople = async (): Promise<Person[]> =>
    await apiCall<Person[]>(`/api/players`, 'GET');

export interface FindProfileProps {
    query: string;
}

export const apiGetResolveProfile = async (
    opts: FindProfileProps
): Promise<Person> =>
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
    ipAddr: string;
    steam_id: SteamID;
    persona_name: string;
    created_on: Date;
    ip_info: PersonIPRecord;
}

export interface PersonMessage {
    person_message_id: bigint;
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

export const apiGetPersonMessages = async (steam_id: SteamID) =>
    await apiCall<PersonMessage[]>(`/api/messages/${steam_id}`, 'GET');
