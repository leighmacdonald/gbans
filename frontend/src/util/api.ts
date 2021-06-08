import { log } from './errors';
import { LogEvent, MsgType } from './game_events';

export enum PermissionLevel {
    Guest = 1,
    Banned = 2,
    Authenticated = 10,
    Moderator = 50,
    Admin = 100
}

export interface apiResponse<T> {
    status: boolean;
    resp: Response;
    json: T | apiError;
}

export interface apiError {
    error?: string;
}

const apiCall = async <TResponse, TRequestBody = Record<string, unknown>>(
    url: string,
    method: string,
    body?: TRequestBody
): Promise<apiResponse<TResponse>> => {
    const headers: Record<string, string> = {
        'Content-Type': 'application/json; charset=UTF-8'
    };
    const opts: RequestInit = {
        mode: 'cors',
        credentials: 'include',
        method: method.toUpperCase()
    };
    const token = localStorage.getItem('token');
    if (token != '') {
        headers['Authorization'] = `Bearer ${token}`;
    }
    if (method === 'POST' && body) {
        opts['body'] = JSON.stringify(body);
    }
    opts.headers = headers;
    const resp = await fetch(url, opts);
    if (resp.status === 403 && token != '') {
        log('invalid token');
    }
    if (!resp.status) {
        throw apiErr('Invalid response code', resp);
    }
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const json = ((await resp.json()) as TResponse as any).data;
    if (json?.error && json.error !== '') {
        throw apiErr(`Error received: ${json.error}`, resp);
    }
    return { json: json, resp: resp, status: resp.ok };
};

class ApiException extends Error {
    public resp: Response;

    constructor(msg: string, response: Response) {
        super(msg);
        this.resp = response;
    }
}

const apiErr = (msg: string, resp: Response): ApiException => {
    return new ApiException(msg, resp);
};

export enum PayloadType {
    okType,
    errType,
    authType,
    authOKType,
    logType,
    logQueryOpts,
    logQueryResults
}

// Used for setting filtering / query options for realtime log event streams
export interface LogQueryOpts {
    log_types: MsgType[];
    limit: number;
    order_desc: boolean;
    query: string;
    source_id: string;
    target_id: string;
    servers: number[];
    result_cache: LogEvent[];
}

export const encode = (t: PayloadType, o: unknown): WebSocketPayload => {
    return {
        payload_type: t,
        data: o
    };
};

export interface WebSocketPayload<TRecord = unknown> {
    payload_type: PayloadType;
    data: TRecord;
}

export interface WebSocketAuthResp {
    status: boolean;
    message: string;
}

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

export interface Server {
    server_id: number;
    server_name: string;
    token: string;
    address: string;
    port: number;
    rcon: string;
    reserved_slots: number;
    password: string;
    token_created_on: Date;
    created_on: Date;
    updated_on: Date;
}

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

export interface QueryFilterProps {
    offset: number;
    limit: number;
    sort_desc: boolean;
    query: string;
    order_by: string;
}

export type IAPIResponseBans = BannedPerson[];

export interface IAPIBanRecord {
    ban_id: number;
    net_id: number;
    steam_id: string;
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

export interface DatabaseStats {
    bans: number;
    bans_day: number;
    bans_week: number;
    bans_month: number;
    bans_3month: number;
    bans_6month: number;
    bans_year: number;
    bans_cidr: number;
    appeals_open: number;
    appeals_closed: number;
    filtered_words: number;
    servers_alive: number;
    servers_total: number;
}

export const apiGetStats = async (): Promise<DatabaseStats> => {
    const resp = await apiCall<DatabaseStats>(`/api/stats`, 'GET');
    return resp.json as DatabaseStats;
};

export interface BanPayload {
    steam_id: string;
    duration: string;
    ban_type: number;
    reason: number;
    reason_text: string;
    network: string;
}

export interface PlayerProfile {
    player: Person;
    friends: Person[];
}

export const apiGetBans = async (): Promise<IAPIBanRecord[] | apiError> => {
    const resp = await apiCall<IAPIResponseBans, QueryFilterProps>(
        `/api/bans`,
        'POST'
    );
    return ((resp.json as IAPIResponseBans) ?? []).map((b): IAPIBanRecord => {
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
    const resp = await apiCall<BannedPerson>(`/api/ban/${ban_id}`, 'GET');
    return resp.json;
};

export const apiCreateBan = async (p: BanPayload): Promise<Ban | apiError> => {
    const resp = await apiCall<Ban, BanPayload>(`/api/ban`, 'POST', p);
    return resp.json;
};

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

export const apiGetServers = async (): Promise<Server[] | apiError> => {
    const resp = await apiCall<Server[]>(`/api/servers`, 'GET');
    return resp.json;
};

export const apiGetPeople = async (): Promise<Person[] | apiError> => {
    const resp = await apiCall<Person[]>(`/api/players`, 'GET');
    return resp.json;
};

export const handleOnLogin = (): void => {
    const r = `${window.location.protocol}//${window.location.hostname}/auth/callback?return_url=${window.location.pathname}`;
    const oid =
        'https://steamcommunity.com/openid/login' +
        '?openid.ns=' +
        encodeURIComponent('http://specs.openid.net/auth/2.0') +
        '&openid.mode=checkid_setup' +
        '&openid.return_to=' +
        encodeURIComponent(r) +
        `&openid.realm=` +
        encodeURIComponent(
            `${window.location.protocol}//${window.location.hostname}`
        ) +
        '&openid.ns.sreg=' +
        encodeURIComponent('http://openid.net/extensions/sreg/1.1') +
        '&openid.claimed_id=' +
        encodeURIComponent(
            'http://specs.openid.net/auth/2.0/identifier_select'
        ) +
        '&openid.identity=' +
        encodeURIComponent(
            'http://specs.openid.net/auth/2.0/identifier_select'
        );
    window.open(oid, '_self');
};

export const handleOnLogout = (): void => {
    localStorage.removeItem('token');
    location.reload();
};
