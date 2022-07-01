import { apiCall, PermissionLevel } from './common';
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

export interface UserProfile {
    steam_id: string;
    created_on: Date;
    updated_on: Date;
    permission_level: PermissionLevel;
    discord_id: string;
    name: string;
    avatar: string;
    avatarfull: string;
    ban_id: number;
}

export interface Person {
    // PlayerSummaries shape
    steamid: string;
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
    created_on: Date;
    updated_on: Date;
}

export interface PlayerProfile {
    player: Person;
    friends: Person[];
}

export const apiGetProfile = async (query: SteamID): Promise<PlayerProfile> => {
    return await apiCall<PlayerProfile>(`/api/profile?query=${query}`, 'GET');
};

export const apiGetCurrentProfile = async (): Promise<UserProfile> => {
    return await apiCall<UserProfile>(`/api/current_profile`, 'GET');
};

export const apiGetPeople = async (): Promise<Person[]> => {
    return await apiCall<Person[]>(`/api/players`, 'GET');
};

export interface FindProfileProps {
    query: string;
}

export const apiGetResolveProfile = async (
    opts: FindProfileProps
): Promise<Person> => {
    return await apiCall<Person, FindProfileProps>(
        '/api/resolve_profile',
        'POST',
        opts
    );
};
