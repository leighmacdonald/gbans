import type { AnticheatQuery, StacEntry } from "../schema/anticheat.ts";
import { transformCreatedOnDate } from "../util/time.ts";
import { apiCall } from "./common.ts";

export const apiGetAnticheatLogs = async (signal: AbortSignal, query: AnticheatQuery) => {
	return (await apiCall<StacEntry[]>(signal, `/api/anticheat/entries`, "GET", query)).map(transformCreatedOnDate);
};
