import { LazyResult } from '../util/table.ts';
import { apiCall, QueryFilter, transformCreatedOnDate } from './common';
import { Asset } from './media';

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
    asset: Asset;
}

export interface DemoQueryFilter extends QueryFilter<DemoFile> {
    steam_id: string;
    map_name: string;
    server_ids: number[];
}

export const apiGetDemos = async (opts: DemoQueryFilter, abortController?: AbortController) => {
    const resp = await apiCall<LazyResult<DemoFile>, DemoQueryFilter>('/api/demos', 'POST', opts, abortController);
    resp.data = resp.data.map(transformCreatedOnDate);
    return resp;
};
