import { apiCall, DataCount } from './common';

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
    return await apiCall<LazyResult<WeaponsOverallResult>>(
        `/api/stats/weapons`,
        'GET'
    );
};

export const apiGetPlayerWeaponsOverall = async (steam_id: string) => {
    return await apiCall<LazyResult<WeaponsOverallResult>>(
        `/api/stats/player/${steam_id}/weapons`,
        'GET'
    );
};
export const apiGetPlayersOverall = async () => {
    return await apiCall<LazyResult<PlayerWeaponStats>>(
        `/api/stats/players`,
        'GET'
    );
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

export interface GamePlayerClass {
    player_class_id: number;
    class_name: string;
    class_key: string;
}

export interface PlayerClassOverallResult
    extends GamePlayerClass,
        MatchPlayerClassStats {}

export interface MatchPlayerClassStats {
    kills: number;
    ka: number;
    assists: number;
    deaths: number;
    kd: number;
    kad: number;
    dpm: number;
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

export interface HealingStats {
    healing: number;
    drops: number;
    near_full_charge_death: number;
    avg_uber_len: number;
    biggest_adv_lost: number;
    major_adv_lost: number;
    charges_uber: number;
    charges_kritz: number;
    charges_vacc: number;
    charges_quick_fix: number;
}

export interface PlayerOverallResult
    extends HealingStats,
        MatchPlayerClassStats {
    buildings: number;
    extinguishes: number;
    health_packs: number;
    shots: number;
    hits: number;
    accuracy: number;
    airshots: number;
    backstabs: number;
    headshots: number;
    healing_taken: number;
    wins: number;
    matches: number;
    win_rate: number;
}

export const apiGetPlayerStats = async (steam_id: string) => {
    return await apiCall<PlayerOverallResult>(
        `/api/stats/player/${steam_id}/overall`,
        'GET'
    );
};

export interface LazyResult<T> extends DataCount {
    data: T[];
}

export const apiGetPlayerClassOverallStats = async (steam_id: string) => {
    return await apiCall<LazyResult<PlayerClassOverallResult>>(
        `/api/stats/player/${steam_id}/classes`,
        'GET'
    );
};

export interface PlayerWeaponStats extends BaseWeaponStats {
    steam_id: string;
    personaname: string;
    avatar_hash: string;
    rank: number;
}

export interface PlayerWeaponStatsResponse
    extends LazyResult<PlayerWeaponStats> {
    weapon: Weapon;
}

export const apiGetPlayerWeaponStats = async (weapon_id: number) => {
    return await apiCall<PlayerWeaponStatsResponse>(
        `/api/stats/weapon/${weapon_id}`,
        'GET'
    );
};
