import type {
	DatabaseStats,
	HealingOverallResult,
	MapUseDetail,
	PlayerClassOverallResult,
	PlayerOverallResult,
	PlayerWeaponStats,
	Weapon,
	WeaponsOverallResult,
} from "../schema/stats.ts";
import type { LazyResult } from "../util/table.ts";
import { apiCall } from "./common";

export const apiGetStats = async (signal: AbortSignal) => await apiCall<DatabaseStats>(signal, `/api/stats`);

export const apiGetMapUsage = async (signal: AbortSignal) => {
	return await apiCall<MapUseDetail[]>(signal, `/api/stats/map`);
};

export const apiGetWeaponsOverall = async (signal: AbortSignal) => {
	return await apiCall<LazyResult<WeaponsOverallResult>>(signal, `/api/stats/weapons`);
};

export const apiGetPlayerWeaponsOverall = async (steam_id: string, signal: AbortSignal) => {
	return await apiCall<LazyResult<WeaponsOverallResult>>(signal, `/api/stats/player/${steam_id}/weapons`);
};

export const apiGetPlayersOverall = async (signal: AbortSignal) => {
	return await apiCall<LazyResult<PlayerWeaponStats>>(signal, `/api/stats/players`);
};

export const apiGetHealersOverall = async (signal: AbortSignal) => {
	return await apiCall<LazyResult<HealingOverallResult>>(signal, `/api/stats/healers`);
};

export const apiGetPlayerStats = async (steam_id: string, signal: AbortSignal) => {
	return await apiCall<PlayerOverallResult>(signal, `/api/stats/player/${steam_id}/overall`);
};

export const apiGetPlayerClassOverallStats = async (steam_id: string, signal: AbortSignal) => {
	return await apiCall<LazyResult<PlayerClassOverallResult>>(signal, `/api/stats/player/${steam_id}/classes`);
};

export interface PlayerWeaponStatsResponse extends LazyResult<PlayerWeaponStats> {
	weapon: Weapon;
}

export const apiGetPlayerWeaponStats = async (weapon_id: number, signal: AbortSignal) => {
	return await apiCall<PlayerWeaponStatsResponse>(signal, `/api/stats/weapon/${weapon_id}`);
};
