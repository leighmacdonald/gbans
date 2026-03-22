import type { Contest, ContestEntry, VoteResult } from "../schema/contest.ts";
import { transformDateRange, transformTimeStampedDates } from "../util/time.ts";
import { apiCall } from "./common";
import { EmptyUUID } from "./const";

export const apiContestSave = async (contest: Contest, signal: AbortSignal) =>
	(contest?.contest_id ?? EmptyUUID) === EmptyUUID
		? await apiCall<Contest, Contest>(signal, `/api/contests`, "POST", contest)
		: await apiCall<Contest, Contest>(signal, `/api/contests/${contest.contest_id}`, "PUT", contest);

export const apiContests = async (signal: AbortSignal) => {
	const resp = await apiCall<Contest[]>(signal, `/api/contests`, "GET");
	return resp.map(transformDateRange).map(transformTimeStampedDates);
};

export const apiContest = async (contest_id: string, signal: AbortSignal) => {
	const contest = await apiCall<Contest>(signal, `/api/contests/${contest_id}`, "GET");
	return transformDateRange(contest);
};

export const apiContestEntries = async (contest_id: string, signal: AbortSignal) => {
	const entries = await apiCall<ContestEntry[]>(signal, `/api/contests/${contest_id}/entries`, "GET");
	return entries.map(transformTimeStampedDates);
};

export const apiContestDelete = async (contest_id: string, signal: AbortSignal) =>
	await apiCall<Contest>(signal, `/api/contests/${contest_id}`, "DELETE");

export const apiContestEntryDelete = async (contest_entry_id: string, signal: AbortSignal) =>
	await apiCall(signal, `/api/contest_entry/${contest_entry_id}`, "DELETE");

export const apiContestEntrySave = async (
	contest_id: string,
	description: string,
	asset_id: string,
	signal: AbortSignal,
) =>
	await apiCall<ContestEntry>(signal, `/api/contests/${contest_id}/submit`, "POST", {
		description,
		asset_id,
	});

export const apiContestEntryVote = async (
	contest_id: string,
	contest_entry_id: string,
	upvote: boolean,
	signal: AbortSignal,
) =>
	await apiCall<VoteResult>(
		signal,
		`/api/contests/${contest_id}/vote/${contest_entry_id}/${upvote ? "up" : "down"}`,
		"GET",
	);
