import { PickupItem, PlayerClass, Team, Weapon } from './const';
import { Person } from './profile';
import { apiCall, Pos } from './common';

export interface Server {
    server_id: number;
    server_name: string;
    server_name_long: string;
    address: string;
    port: number;
    password_protected: boolean;
    vac: boolean;
    region: string;
    cc: string;
    latitude: number;
    longitude: number;
    current_map: string;
    tags: string[];
    default_map: string;
    reserved_slots: number;
    players_max: number;
    players: PlayerInfo[];
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
    extra?: string;
    team: Team;
    created_on: string;
    meta_data?: object;
}

// Used for setting filtering / query options for realtime log event streams
export interface LogQueryOpts {
    log_types?: number[];
    limit?: number;
    order_desc?: boolean;
    query?: string;
    source_id?: string;
    target_id?: string;
    servers?: number[];
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

export const apiGetServers = async (): Promise<Server[]> => {
    return await apiCall<Server[]>(`/api/servers`, 'GET');
};
