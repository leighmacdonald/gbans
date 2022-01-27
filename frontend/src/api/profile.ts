import { apiCall, apiError } from './common';

export enum profileState {
    Incomplete = 0,
    Setup = 1
}

export enum communityVisibilityState {
    Private = 1,
    FriendOnly = 2,
    Public = 3
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

    // Custom attributes
    steam_id: string;
    ip_addr: string;
    created_on: Date;
    updated_on: Date;
}

export interface PlayerProfile {
    player: Person;
    friends: Person[];
}

export const apiGetProfile = async (
    query: string
): Promise<PlayerProfile | apiError> => {
    const resp = await apiCall<PlayerProfile>(
        `/api/profile?query=${query}`,
        'GET'
    );
    return resp.json;
};

export const apiGetCurrentProfile = async (): Promise<
    PlayerProfile | apiError
> => {
    const resp = await apiCall<PlayerProfile>(`/api/current_profile`, 'GET');
    return resp.json;
};

export const apiGetPeople = async (): Promise<Person[] | apiError> => {
    const resp = await apiCall<Person[]>(`/api/players`, 'GET');
    return resp.json;
};

export interface FindProfileProps {
    query: string;
}

export const apiGetResolveProfile = async (
    opts: FindProfileProps
): Promise<Person> => {
    const resp = await apiCall<Person, FindProfileProps>(
        '/api/resolve_profile',
        'POST',
        opts
    );
    return resp.json;
};
