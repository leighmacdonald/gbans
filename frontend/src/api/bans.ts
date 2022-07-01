import { apiCall, apiError, QueryFilterProps } from './common';
import { communityVisibilityState, Person, profileState } from './profile';
import { SteamID } from './const';

export interface ChatMessage {
    message: string;
    created_on: Date;
}

export interface BannedPerson {
    ban: Ban;
    person: Person;
    history_chat: ChatMessage[];
    history_personaname: string[];
    history_connections: string[];
    history_ip: string[];
}

export interface Ban {
    ban_id: number;
    net_id: number;
    steam_id: number;
    cidr: string;
    author_id: number;
    ban_type: number;
    reason: number;
    reason_text: string;
    note: string;
    source: number;
    valid_until: Date;
    created_on: Date;
    updated_on: Date;
}

export type IAPIResponseBans = BannedPerson[];

export interface IAPIBanRecord {
    ban_id: number;
    net_id: number;
    steam_id: SteamID;
    cidr: string;
    author_id: number;
    ban_type: number;
    reason: number;
    reason_text: string;
    note: string;
    source: number;
    valid_until: Date;
    created_on: Date;
    updated_on: Date;

    steamid: string;
    communityvisibilitystate: communityVisibilityState;
    profilestate: profileState;
    personaname: string;
    profileurl: string;
    avatar: string;
    avatarmedium: string;
    avatarfull: string;
    personastate: number;
    realname: string;
    timecreated: number;
    personastateflags: number;
    loccountrycode: string;

    // Custom attributes
    ip_addr: string;
}

export interface BanPayload {
    steam_id: string;
    duration: string;
    ban_type: number;
    reason: number;
    reason_text: string;
    network: string;
}

export const apiGetBans = async (): Promise<IAPIBanRecord[]> => {
    const resp = await apiCall<IAPIResponseBans, QueryFilterProps>(
        `/api/bans`,
        'POST'
    );
    return (resp ?? []).map((b): IAPIBanRecord => {
        return {
            author_id: b.ban.author_id,
            avatar: b.person.avatar,
            avatarfull: b.person.avatarfull,
            avatarmedium: b.person.avatarmedium,
            ban_id: b.ban.ban_id,
            ban_type: b.ban.ban_type,
            cidr: b.ban.cidr,
            communityvisibilitystate: b.person.communityvisibilitystate,
            created_on: b.ban.created_on,
            ip_addr: b.person.ip_addr,
            loccountrycode: b.person.loccountrycode,
            net_id: b.ban.net_id,
            note: b.ban.note,
            personaname: b.person.personaname,
            personastate: b.person.personastate,
            personastateflags: b.person.personastateflags,
            profilestate: b.person.profilestate,
            profileurl: b.person.profileurl,
            realname: b.person.realname,
            reason: b.ban.reason,
            reason_text: b.ban.reason_text,
            source: b.ban.source,
            steam_id: b.person.steam_id,
            steamid: b.person.steamid,
            timecreated: b.person.timecreated,
            updated_on: b.ban.updated_on,
            valid_until: b.ban.valid_until
        };
    });
};

export const apiGetBan = async (
    ban_id: number
): Promise<BannedPerson | apiError> => {
    return await apiCall<BannedPerson>(`/api/ban/${ban_id}`, 'GET');
};

export const apiCreateBan = async (p: BanPayload): Promise<Ban | apiError> => {
    return await apiCall<Ban, BanPayload>(`/api/ban`, 'POST', p);
};
