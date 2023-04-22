import { apiCall } from './common';

export const wordFilterSeparator = '---';

export interface Filter {
    filter_id?: number;
    author_id?: bigint;
    pattern: RegExp | string;
    is_regex: boolean;
    is_enabled?: boolean;
    trigger_count?: number;
    created_on?: Date;
    updated_on?: Date;
}

export const apiGetFilters = async () =>
    await apiCall<Filter[]>(`/api/filters`, 'GET');

export const apiSaveFilter = async (filter: Filter) =>
    await apiCall<Filter>(`/api/filters`, 'POST', filter);

export interface FilterQuery {
    query: string;
    corpus: string;
}

export const apiMatchFilter = async (opts: FilterQuery) =>
    await apiCall<Filter[]>(`/api/filter_match`, 'POST', opts);

export const apiDeleteFilter = async (word_id: number) =>
    await apiCall(`/api/filters/${word_id}`, 'DELETE');
