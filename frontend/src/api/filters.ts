import { LazyResult } from '../component/table/LazyTableSimple';
import { apiCall, QueryFilter } from './common';

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

export interface FiltersQueryFilter extends QueryFilter<Filter> {}

export const apiGetFilters = async (
    opts: FiltersQueryFilter,
    abortController?: AbortController
) =>
    await apiCall<LazyResult<Filter>>(
        `/api/filters/query`,
        'POST',
        opts,
        abortController
    );

export const apiSaveFilter = async (filter: Filter) =>
    await apiCall<Filter>(`/api/filters`, 'POST', filter);

export interface FilterQuery {
    query: string;
    corpus: string;
}

export const apiDeleteFilter = async (word_id: number) =>
    await apiCall(`/api/filters/${word_id}`, 'DELETE');
