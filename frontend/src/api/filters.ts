import type { Filter, WarningState } from "../schema/filters.ts";
import { transformCreatedOnDate } from "../util/time.ts";
import { apiCall } from "./common";

export const apiGetFilters = async (abortController?: AbortController) =>
	await apiCall<Filter[]>(`/api/filters`, "GET", abortController);

export const apiCreateFilter = async (filter: Filter) => await apiCall<Filter>(`/api/filters`, "POST", filter);

export const apiEditFilter = async (filter_id: number, filter: Filter) =>
	await apiCall<Filter>(`/api/filters/${filter_id}`, "POST", filter);

export const apiDeleteFilter = async (word_id: number) => await apiCall(`/api/filters/${word_id}`, "DELETE");

export const apiGetWarningState = async (abortController?: AbortController) => {
	const resp = await apiCall<WarningState>("/api/filters/state", "GET", undefined, abortController);

	resp.current = resp.current.map(transformCreatedOnDate);

	return resp;
};
