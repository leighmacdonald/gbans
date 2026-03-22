import type { VoteQueryFilter, VoteResult } from "../schema/votes.ts";
import type { LazyResult } from "../util/table.ts";
import { transformCreatedOnDate } from "../util/time.ts";
import { apiCall } from "./common.ts";

export const apiVotesQuery = async (opts: VoteQueryFilter, signal: AbortSignal) => {
	const resp = await apiCall<LazyResult<VoteResult>>(signal, "/api/votes", "POST", opts);
	resp.data = resp.data.map(transformCreatedOnDate);

	return resp;
};
