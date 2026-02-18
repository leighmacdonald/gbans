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

export const apiGetStats = async () => await apiCall<DatabaseStats>(`/api/stats`, "GET");

export const apiGetMapUsage = async () => {
	return await apiCall<MapUseDetail[]>(`/api/stats/map`, "GET");
};

export const apiGetWeaponsOverall = async () => {
	return await apiCall<LazyResult<WeaponsOverallResult>>(`/api/stats/weapons`, "GET");
};

export const apiGetPlayerWeaponsOverall = async (steam_id: string) => {
	return await apiCall<LazyResult<WeaponsOverallResult>>(`/api/stats/player/${steam_id}/weapons`, "GET");
};

export const apiGetPlayersOverall = async () => {
	return await apiCall<LazyResult<PlayerWeaponStats>>(`/api/stats/players`, "GET");
};

export const apiGetHealersOverall = async (abortController?: AbortController) => {
	return await apiCall<LazyResult<HealingOverallResult>>(`/api/stats/healers`, "GET", undefined, abortController);
};

export const apiGetPlayerStats = async (steam_id: string, abortController: AbortController) => {
	return await apiCall<PlayerOverallResult>(
		`/api/stats/player/${steam_id}/overall`,
		"GET",
		undefined,
		abortController,
	);
};

export const apiGetPlayerClassOverallStats = async (steam_id: string) => {
	return await apiCall<LazyResult<PlayerClassOverallResult>>(`/api/stats/player/${steam_id}/classes`, "GET");
};

export interface PlayerWeaponStatsResponse extends LazyResult<PlayerWeaponStats> {
	weapon: Weapon;
}

export const apiGetPlayerWeaponStats = async (weapon_id: number) => {
	return await apiCall<PlayerWeaponStatsResponse>(`/api/stats/weapon/${weapon_id}`, "GET");
};
