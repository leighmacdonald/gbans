import type { Contest, ContestEntry, VoteResult } from "../schema/contest.ts";
import { transformDateRange, transformTimeStampedDates } from "../util/time.ts";
import { apiCall } from "./common";
import { EmptyUUID } from "./const";

export const apiContestSave = async (contest: Contest) =>
	(contest?.contest_id ?? EmptyUUID) === EmptyUUID
		? await apiCall<Contest, Contest>(`/api/contests`, "POST", contest)
		: await apiCall<Contest, Contest>(`/api/contests/${contest.contest_id}`, "PUT", contest);

export const apiContests = async () => {
	const resp = await apiCall<Contest[]>(`/api/contests`, "GET");
	return resp.map(transformDateRange).map(transformTimeStampedDates);
};

export const apiContest = async (contest_id: string) => {
	const contest = await apiCall<Contest>(`/api/contests/${contest_id}`, "GET");
	return transformDateRange(contest);
};

export const apiContestEntries = async (contest_id: string) => {
	const entries = await apiCall<ContestEntry[]>(`/api/contests/${contest_id}/entries`, "GET");
	return entries.map(transformTimeStampedDates);
};

export const apiContestDelete = async (contest_id: string) =>
	await apiCall<Contest>(`/api/contests/${contest_id}`, "DELETE");

export const apiContestEntryDelete = async (contest_entry_id: string) =>
	await apiCall(`/api/contest_entry/${contest_entry_id}`, "DELETE");

export const apiContestEntrySave = async (contest_id: string, description: string, asset_id: string) =>
	await apiCall<ContestEntry>(`/api/contests/${contest_id}/submit`, "POST", {
		description,
		asset_id,
	});

export const apiContestEntryVote = async (contest_id: string, contest_entry_id: string, upvote: boolean) =>
	await apiCall<VoteResult>(`/api/contests/${contest_id}/vote/${contest_entry_id}/${upvote ? "up" : "down"}`, "GET");
