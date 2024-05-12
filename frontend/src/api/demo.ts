import { apiCall, QueryFilter, transformCreatedOnDate } from './common';

export interface DemoFile {
    demo_id: number;
    server_id: number;
    server_name_short: string;
    server_name_long: string;
    title: string;
    created_on: Date;
    size: number;
    downloads: number;
    map_name: string;
    archive: boolean;
    stats: Record<string, object>;
    asset_id: string;
}

export interface DemoQueryFilter extends QueryFilter<DemoFile> {
    steam_id: string;
    map_name: string;
    server_ids: number[];
}

export const apiGetDemos = async (abortController?: AbortController) => {
    const resp = await apiCall<DemoFile[]>('/api/demos', 'POST', undefined, abortController);
    return resp.map(transformCreatedOnDate);
};
