import { apiCall } from './common';

export interface Filter {
    word_id: number;
    patterns: string[];
    created_on?: Date;
    updated_on?: Date;
    discord_id?: string;
    discord_created_on?: Date;
    filter_name: string;
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
