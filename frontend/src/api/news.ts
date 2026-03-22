import type { NewsEntry } from "../schema/news.ts";
import { transformTimeStampedDates, transformTimeStampedDatesList } from "../util/time.ts";
import { apiCall } from "./common";

export const apiGetNewsLatest = async (signal: AbortSignal) =>
	transformTimeStampedDatesList(await apiCall<NewsEntry[]>(signal, `/api/news_latest`));

export const apiGetNewsAll = async (signal: AbortSignal) =>
	transformTimeStampedDatesList(await apiCall<NewsEntry[]>(signal, `/api/news_all`));

export const apiNewsCreate = async (title: string, body_md: string, is_published: boolean, signal: AbortSignal) =>
	transformTimeStampedDates(
		await apiCall<NewsEntry>(signal, `/api/news`, "POST", {
			title,
			body_md,
			is_published: is_published,
		}),
	);

export const apiNewsSave = async (entry: NewsEntry, signal: AbortSignal) =>
	transformTimeStampedDates(
		await apiCall<NewsEntry>(signal, `/api/news/${entry.news_id}`, "POST", {
			title: entry.title,
			body_md: entry.body_md,
			is_published: entry.is_published,
		}),
	);

export const apiNewsDelete = async (news_id: number, signal: AbortSignal) =>
	await apiCall<NewsEntry>(signal, `/api/news/${news_id}`, "DELETE");
