import { apiCall } from './common';

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

export const apiGetStats = async () =>
    await apiCall<DatabaseStats>(`/api/stats`, 'GET');

export interface TeamScores {
    red: number;
    red_time: number;
    blu: number;
    blu_time: number;
}

export interface MapUseDetail {
    map: string;
    playtime: number;
    percent: number;
}

export const apiGetMapUsage = async () => {
    return await apiCall<MapUseDetail[]>(`/api/stats/map`, 'GET');
};

export interface Weapon {
    weapon_id: number;
    key: string;
    name: string;
}

export interface BaseWeaponStats {
    kills: number;
    damage: number;
    headshots: number;
    airshots: number;
    backstabs: number;
    shots: number;
    hits: number;
}

export interface WeaponsOverallResult extends Weapon, BaseWeaponStats {
    kills_pct: number;
    damage_pct: number;
    headshots_pct: number;
    airshots_pct: number;
    backstabs_pct: number;
    shots_pct: number;
    hits_pct: number;
}

export const apiGetWeaponsOverall = async () => {
    return await apiCall<WeaponsOverallResult[]>(`/api/stats/weapons`, 'GET');
};

export const apiGetPlayersOverall = async () => {
    return await apiCall<PlayerWeaponStats[]>(`/api/stats/players`, 'GET');
};

export interface BaseWeaponStats {
    ka: number;
    kills: number;
    assists: number;
    deaths: number;
    kd: number;
    kad: number;
    dpm: number;
    shots: number;
    hits: number;
    accuracy: number;
    airshots: number;
    backstabs: number;
    headshots: number;
    playtime: number;
    dominations: number;
    dominated: number;
    revenges: number;
    damage: number;
    damage_taken: number;
    captures: number;
    captures_blocked: number;
    buildings_destroyed: number;
}

export interface PlayerWeaponStats extends BaseWeaponStats {
    steam_id: string;
    personaname: string;
    avatar_hash: string;
    rank: number;
}

interface PlayerWeaponStatsResponse {
    weapon: Weapon;
    players: PlayerWeaponStats[];
}

export const apiGetPlayerWeaponStats = async (weapon_id: number) => {
    return await apiCall<PlayerWeaponStatsResponse>(
        `/api/stats/weapon/${weapon_id}`,
        'GET'
    );
};
