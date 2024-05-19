import { parseDateTime } from '../util/text.tsx';
import { apiCall, TimeStamped, transformTimeStampedDates } from './common';

export interface BaseServer {
    server_id: number;
    host: string;
    port: number;
    ip: string;
    name: string;
    name_short: string;
    region: string;
    cc: string;
    players: number;
    max_players: number;
    bots: number;
    map: string;
    game_types: string[];
    latitude: number;
    longitude: number;
    distance: number; // calculated on load
}

export const cleanMapName = (name: string): string => {
    if (!name.startsWith('workshop/')) {
        return name;
    }
    const a = name.split('/');
    if (a.length != 2) {
        return name;
    }
    const b = a[1].split('.ugc');
    if (a.length != 2) {
        return name;
    }
    return b[0];
};

export interface ServerSimple {
    server_id: number;
    server_name: string;
    server_name_long: string;
    colour: string;
}

export interface Server extends TimeStamped {
    server_id: number;
    short_name: string;
    name: string;
    address: string;
    port: number;
    password: string;
    rcon: string;
    region: string;
    cc: string;
    latitude: number;
    longitude: number;
    default_map: string;
    reserved_slots: number;
    players_max: number;
    is_enabled: boolean;
    colour: string;
    enable_stats: boolean;
    log_secret: number;
    token_created_on: Date;
}

export interface Location {
    latitude: number;
    longitude: number;
}

interface UserServers {
    servers: BaseServer[];
    lat_long: Location;
}

export const apiGetServerStates = async (abortController?: AbortController) =>
    await apiCall<UserServers>(`/api/servers/state`, 'GET', undefined, abortController);

export interface SaveServerOpts {
    server_name_short: string;
    server_name: string;
    host: string;
    port: number;
    rcon: string;
    password: string;
    reserved_slots: number;
    region: string;
    cc: string;
    lat: number;
    lon: number;
    is_enabled: boolean;
    enable_stats: boolean;
    log_secret: number;
}

export const apiCreateServer = async (opts: SaveServerOpts) =>
    transformTimeStampedDates(await apiCall<Server, SaveServerOpts>(`/api/servers`, 'POST', opts));

export const apiSaveServer = async (server_id: number, opts: SaveServerOpts) => {
    const resp = transformTimeStampedDates(
        await apiCall<Server, SaveServerOpts>(`/api/servers/${server_id}`, 'POST', opts)
    );
    resp.token_created_on = parseDateTime(resp.token_created_on as unknown as string);
    return resp;
};

export const apiGetServersAdmin = async (abortController?: AbortController) => {
    const resp = await apiCall<Server[]>(`/api/servers_admin`, 'POST', undefined, abortController);
    return resp.map(transformTimeStampedDates).map((s) => {
        s.token_created_on = parseDateTime(s.token_created_on as unknown as string);
        return s;
    });
};

export const apiGetServers = async () => apiCall<ServerSimple[]>(`/api/servers`, 'GET', undefined);

export const apiDeleteServer = async (server_id: number) => await apiCall(`/api/servers/${server_id}`, 'DELETE');

export type AuthType = 'steam' | 'name' | 'ip';

export type OverrideType = 'command' | 'group';

export type OverrideAccess = 'allow' | 'deny';

export type SMAdmin = {
    admin_id: number;
    steam_id: string;
    auth_type: AuthType;
    identity: string;
    password: string;
    flags: string;
    name: string;
    immunity: number;
} & TimeStamped;

export type SMGroups = {
    group_id: number;
    flags: string;
    name: string;
    immunity_level: number;
} & TimeStamped;

type Flags =
    | 'z'
    | 'a'
    | 'b'
    | 'c'
    | 'd'
    | 'e'
    | 'f'
    | 'g'
    | 'h'
    | 'i'
    | 'j'
    | 'k'
    | 'l'
    | 'm'
    | 'n'
    | 'o'
    | 'p'
    | 'q'
    | 'r'
    | 's'
    | 't';

export const hasSMFlag = (flag: Flags, entity?: SMGroups | SMAdmin) => {
    return entity?.flags.includes(flag) ?? false;
};

export type SMGroupImmunity = {
    group_id: number;
    other_id: number;
};

export type SMOverrides = {
    type: OverrideType;
    name: string;
    flags: string;
};

export type SMAdminGroups = {
    admin_id: number;
    group_id: number;
    inherit_order: number;
};

export const apiGetSMAdmins = async () =>
    (await apiCall<SMAdmin[]>('/api/smadmin/admins')).map(transformTimeStampedDates);

export const apiCreateSMAdmin = async (
    name: string,
    immunity: number,
    flags: string,
    auth_type: AuthType,
    identity: string,
    password: string
) =>
    transformTimeStampedDates(
        await apiCall<SMAdmin>('/api/smadmin/admins', 'POST', {
            name,
            immunity,
            flags,
            auth_type,
            identity,
            password
        })
    );

export const apiSaveSMAdmin = async (
    admin_id: number,
    name: string,
    immunity: number,
    flags: string,
    auth_type: AuthType,
    identity: string,
    password: string
) =>
    transformTimeStampedDates(
        await apiCall<SMAdmin>(`/api/smadmin/admins/${admin_id}`, 'POST', {
            name,
            immunity,
            flags,
            auth_type,
            identity,
            password
        })
    );

export const apiDeleteSMAdmin = async (admin_id: number) =>
    await apiCall(`/api/smadmin/admins/${admin_id}`, 'DELETE', undefined);

export const apiGetSMGroups = async () =>
    (await apiCall<SMGroups[]>('/api/smadmin/groups')).map(transformTimeStampedDates);

export const apiCreateSMGroup = async (name: string, immunity: number, flags: string) =>
    transformTimeStampedDates(
        await apiCall<SMGroups>('/api/smadmin/groups', 'POST', {
            name,
            immunity,
            flags
        })
    );

export const apiSaveSMGroup = async (group_id: number, name: string, immunity: number, flags: string) =>
    transformTimeStampedDates(
        await apiCall<SMGroups>(`/api/smadmin/groups/${group_id}`, 'POST', {
            name,
            immunity,
            flags
        })
    );

export const apiDeleteSMGroup = async (group_id: number) =>
    await apiCall(`/api/smadmin/groups/${group_id}`, 'DELETE', undefined);
