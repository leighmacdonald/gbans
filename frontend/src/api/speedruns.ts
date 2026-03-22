import type { SpeedrunMapOverview, SpeedrunResult } from "../schema/speedrun.ts";
import { transformCreatedOnDate } from "../util/time.ts";
import { apiCall } from "./common.ts";

export const getSpeedrunsTopOverall = async (count: number = 5, signal: AbortSignal) => {
	const results = await apiCall<Record<string, SpeedrunResult[]>>(
		signal,
		`/api/speedruns/overall/top?count=${count}`,
	);
	for (const key of Object.keys(results)) {
		results[key] = results[key].map(transformCreatedOnDate);
	}
	return results;
};

export const getSpeedrunsTopMap = async (map_name: string, signal: AbortSignal) => {
	return (await apiCall<SpeedrunMapOverview[]>(signal, `/api/speedruns/map?map_name=${map_name}`)).map(
		transformCreatedOnDate,
	);
};

export const getSpeedrunsRecent = async (count: number = 5, signal: AbortSignal) => {
	return (await apiCall<SpeedrunMapOverview[]>(signal, `/api/speedruns/overall/recent?count=${count}`)).map(
		transformCreatedOnDate,
	);
};

export const getSpeedrun = async (speedrun_id: number, signal: AbortSignal): Promise<SpeedrunResult> => {
	const r = transformCreatedOnDate(await apiCall<SpeedrunResult>(signal, `/api/speedruns/byid/${speedrun_id}`));
	r.players = r.players.sort((a, b) => {
		return a.duration > b.duration ? 1 : -1;
	});
	r.point_captures.map((p) => {
		p.players = p.players.sort((a, b) => {
			return a.duration > b.duration ? 1 : -1;
		});
		return p;
	});
	return r;
};
