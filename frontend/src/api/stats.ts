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

export interface WeaponsOverallResult extends Weapon {
    kills: number;
    kills_pct: number;
    damage: number;
    damage_pct: number;
    headshots: number;
    headshots_pct: number;
    airshots: number;
    airshots_pct: number;
    backstabs: number;
    backstabs_pct: number;
    shots: number;
    shots_pct: number;
    hits: number;
    hits_pct: number;
}

export const apiGetWeaponsOverall = async () => {
    return await apiCall<WeaponsOverallResult[]>(`/api/stats/weapons`, 'GET');
};
