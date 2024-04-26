import { LazyResult } from '../util/table.ts';
import {
    apiCall,
    QueryFilter,
    TimeStamped,
    transformTimeStampedDates
} from './common';

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
    await apiCall<UserServers>(
        `/api/servers/state`,
        'GET',
        undefined,
        abortController
    );

export interface SaveServerOpts {
    server_name_short: string;
    server_name: string;
    host: string;
    port: number;
    rcon: string;
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
    transformTimeStampedDates(
        await apiCall<Server, SaveServerOpts>(`/api/servers`, 'POST', opts)
    );

export const apiSaveServer = async (server_id: number, opts: SaveServerOpts) =>
    transformTimeStampedDates(
        await apiCall<Server, SaveServerOpts>(
            `/api/servers/${server_id}`,
            'POST',
            opts
        )
    );

export interface ServerQueryFilter extends QueryFilter<Server> {
    include_disabled?: boolean;
}

export const apiGetServersAdmin = async (
    opts: ServerQueryFilter,
    abortController?: AbortController
) =>
    await apiCall<LazyResult<Server>>(
        `/api/servers_admin`,
        'POST',
        opts,
        abortController
    );

export const apiGetServers = async () =>
    apiCall<ServerSimple[]>(`/api/servers`, 'GET', undefined);

export const apiDeleteServer = async (server_id: number) =>
    await apiCall(`/api/servers/${server_id}`, 'DELETE');

export interface SlimServer {
    addr: string;
    name: string;
    region: number;
    players: number;
    max_players: number;
    bots: number;
    map: string;
    game_types: string[];
    latitude: number;
    longitude: number;
    distance: number;
}
