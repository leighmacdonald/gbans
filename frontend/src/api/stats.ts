import { apiCall } from './common';

export interface CommonStats {
    kills: number;
    assists: number;
    damage: number;
    healing: number;
    shots: number;
    hits: number;
    suicides: number;
    extinguishes: number;
    point_captures: number;
    point_defends: number;
    medic_dropped_uber: number;
    object_built: number;
    object_destroyed: number;
    messages: number;
    messages_team: number;
    pickup_ammo_large: number;
    pickup_ammo_medium: number;
    pickup_ammo_small: number;
    pickup_hp_large: number;
    pickup_hp_medium: number;
    pickup_hp_small: number;
    spawn_scout: number;
    spawn_soldier: number;
    spawn_pyro: number;
    spawn_demo: number;
    spawn_heavy: number;
    spawn_engineer: number;
    spawn_medic: number;
    spawn_spy: number;
    spawn_sniper: number;
    dominations: number;
    revenges: number;
    playtime: number;
}

export interface PlayerStats extends CommonStats {
    deaths: number;
    games: number;
    wins: number;
    losses: number;
    damage_taken: number;
    dominated: number;
}

export interface GlobalStats extends CommonStats {
    unique_players: number;
}

export type ServerStats = CommonStats;
export type MapStats = CommonStats;

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
