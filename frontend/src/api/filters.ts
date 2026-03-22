import type { Filter, WarningState } from "../schema/filters.ts";
import { transformCreatedOnDate } from "../util/time.ts";
import { apiCall } from "./common";

export const apiGetFilters = async (signal: AbortSignal) => await apiCall<Filter[]>(signal, `/api/filters`, "GET");

export const apiCreateFilter = async (filter: Filter, signal: AbortSignal) =>
	await apiCall<Filter>(signal, `/api/filters`, "POST", filter);

export const apiEditFilter = async (filter_id: number, filter: Filter, signal: AbortSignal) =>
	await apiCall<Filter>(signal, `/api/filters/${filter_id}`, "POST", filter);

export const apiDeleteFilter = async (word_id: number, signal: AbortSignal) =>
	await apiCall(signal, `/api/filters/${word_id}`, "DELETE");

export const apiGetWarningState = async (signal: AbortSignal) => {
	const resp = await apiCall<WarningState>(signal, "/api/filters/state", "GET");

	resp.current = resp.current.map(transformCreatedOnDate);

	return resp;
};
