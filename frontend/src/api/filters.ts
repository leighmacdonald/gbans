import { transformCreatedOnDate } from '../util/time.ts';
import { BanReason } from './bans';
import { apiCall } from './common';

export enum FilterAction {
    Kick,
    Mute,
    Ban
}

export const FilterActionCollection = [FilterAction.Kick, FilterAction.Mute, FilterAction.Ban];

export const filterActionString = (fa: FilterAction) => {
    switch (fa) {
        case FilterAction.Ban:
            return 'Ban';
        case FilterAction.Kick:
            return 'Kick';
        case FilterAction.Mute:
            return 'Mute';
    }
};

export interface Filter {
    filter_id?: number;
    author_id?: bigint;
    pattern: RegExp | string;
    is_regex: boolean;
    is_enabled?: boolean;
    trigger_count?: number;
    action: FilterAction;
    duration: string;
    weight: number;
    created_on?: Date;
    updated_on?: Date;
}

export const apiGetFilters = async (abortController?: AbortController) =>
    await apiCall<Filter[]>(`/api/filters`, 'GET', abortController);

export const apiCreateFilter = async (filter: Filter) => await apiCall<Filter>(`/api/filters`, 'POST', filter);

export const apiEditFilter = async (filter_id: number, filter: Filter) =>
    await apiCall<Filter>(`/api/filters/${filter_id}`, 'POST', filter);

export const apiDeleteFilter = async (word_id: number) => await apiCall(`/api/filters/${word_id}`, 'DELETE');

export interface UserWarning {
    warn_reason: BanReason;
    message: string;
    matched: string;
    matched_filter: Filter;
    created_on: Date;
    personaname: string;
    avatar: string;
    server_name: string;
    server_id: number;
    steam_id: string;
    current_total: number;
}

export interface warningState {
    max_weight: number;
    current: UserWarning[];
}

export const apiGetWarningState = async (abortController?: AbortController) => {
    const resp = await apiCall<warningState>('/api/filters/state', 'GET', undefined, abortController);

    resp.current = resp.current.map(transformCreatedOnDate);

    return resp;
};
