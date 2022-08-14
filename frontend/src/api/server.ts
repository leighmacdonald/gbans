import { PickupItem, PlayerClass, Team, Weapon } from './const';
import { Person } from './profile';
import { apiCall, Pos } from './common';
import { LatLngLiteral } from 'leaflet';
import SteamID from 'steamid';

export interface ServerState {
    server_id: number;
    name: string;
    name_short: string;
    host: string;
    port: number;
    enabled: boolean;
    region: string;
    cc: string;
    location: LatLngLiteral;
    reserved: number;
    last_update: string;
    name_a2s: string;
    protocol: number;
    map: string;
    folder: string;
    game: string;
    app_id: number;
    player_count: number;
    max_players: number;
    bots: number;
    server_type: string;
    server_os: string;
    password: boolean;
    vac: boolean;
    version: string;
    steam_id: string;
    keywords: string[];
    game_id: number;
    stv_port: number;
    stv_name: string;
    players: ServerStatePlayer[];
}

export interface ServerStatePlayer {
    user_id: number;
    name: string;
    steam_id: string;
    connected_time: number;
    state: string;
    ping: number;
}

export interface Server {
    server_id: number;
    server_name: string;
    server_name_long: string;
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
    created_on: Date;
    updated_on: Date;
}

export interface PlayerInfo {
    steam_id: number;
    name: string;
    user_id: number;
    connected_time: number;
}

export interface ServerEvent {
    log_id: number;
    server: Server;
    event_type: number;
    source?: Person;
    target?: Person;
    player_class: PlayerClass;
    weapon: Weapon;
    damage: number;
    healing: number;
    item: PickupItem;
    attacker_pos?: Pos;
    victim_pos?: Pos;
    assister_pos?: Pos;
    team: Team;
    created_on: string;
    meta_data: Record<string, unknown>;
}

// Used for setting filtering / query options for realtime log event streams
export interface LogQueryOpts {
    log_types?: number[];
    limit?: number;
    order_desc?: boolean;
    query?: string;
    source_id?: SteamID;
    target_id?: string;
    servers?: number[];
    sent_after?: Date;
    sent_before?: Date;
    network?: string;
}

export const findLogs = async (opts: LogQueryOpts): Promise<ServerEvent[]> => {
    if (opts.servers?.length == 1 && opts.servers[0] == 0) {
        // 0 is equivalent to all servers.
        opts.servers = [];
    }
    return await apiCall<ServerEvent[], LogQueryOpts>(
        `/api/events`,
        'POST',
        opts
    );
};

export const apiGetServerStates = async () =>
    await apiCall<ServerState[]>(`/api/servers/state`, 'GET');

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
}

export const apiCreateServer = async (opts: SaveServerOpts) =>
    await apiCall<Server, SaveServerOpts>(`/api/servers`, 'POST', opts);

export const apiSaveServer = async (server_id: number, opts: SaveServerOpts) =>
    await apiCall<Server, SaveServerOpts>(
        `/api/servers/${server_id}`,
        'POST',
        opts
    );

export const apiGetServers = async () =>
    await apiCall<Server[]>(`/api/servers`, 'GET');

export const apiDeleteServer = async (server_id: number) =>
    await apiCall(`/api/servers/${server_id}`, 'DELETE');
