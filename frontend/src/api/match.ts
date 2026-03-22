import type { MatchesQueryOpts, MatchResult, MatchSummary } from "../schema/stats.ts";
import type { LazyResult } from "../util/table.ts";
import { parseDateTime } from "../util/time.ts";
import { apiCall } from "./common";

export const transformMatchDates = (item: MatchResult) => {
	item.time_start = parseDateTime(item.time_start as unknown as string);
	item.time_end = parseDateTime(item.time_end as unknown as string);
	item.players = item.players.map((t) => {
		t.time_start = parseDateTime(t.time_start as unknown as string);
		t.time_end = parseDateTime(t.time_end as unknown as string);
		return t;
	});
	return item;
};

export const apiGetMatch = async (match_id: string, signal: AbortSignal) => {
	const match = await apiCall<MatchResult>(signal, `/api/log/${match_id}`);
	return transformMatchDates(match);
};

export const apiGetMatches = async (opts: MatchesQueryOpts, signal: AbortSignal) => {
	const resp = await apiCall<LazyResult<MatchSummary>, MatchesQueryOpts>(signal, `/api/logs`, "POST", opts);
	resp.data = resp.data.map((m) => {
		m.time_start = parseDateTime(m.time_start as unknown as string);
		m.time_end = parseDateTime(m.time_end as unknown as string);
		return m;
	});

	return resp;
};
