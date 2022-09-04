import { apiCall } from './common';

export interface Filter {
    word_id: number;
    patterns: string[];
    created_on: Date;
    updated_on?: Date;
    discord_id: string;
    discord_created_on: Date;
    filter_name: string;
}

export const apiGetFilters = async () =>
    await apiCall<Filter[]>(`/api/filters`, 'GET');
