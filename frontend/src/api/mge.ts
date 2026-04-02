import type { MGEHistory, MGEStat, QueryMGE, QueryMGEHistory } from "../schema/mge";
import type { LazyResult } from "../util/table";
import { parseDateTime } from "../util/time";
import { apiCall } from "./common";

export const apiMGEOverall = async (signal: AbortSignal, opts: QueryMGE) => {
	const response = await apiCall<LazyResult<MGEStat>>(signal, "/api/mge/ratings/overall", "GET", opts);
	response.data = response.data.map((s) => ({ ...s, lastplayed: parseDateTime(s.last_played as unknown as string) }));
	return response;
};

export const apiMGEHistory = async (signal: AbortSignal, opts: QueryMGEHistory) => {
	const response = await apiCall<LazyResult<MGEHistory>>(signal, "/api/mge/history", "GET", opts);
	response.data = response.data.map((s) => ({ ...s, game_time: parseDateTime(s.game_time as unknown as string) }));
	return response;
};
