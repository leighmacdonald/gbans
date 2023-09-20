import { apiCall } from './common';
import { parseDateTime } from '../util/text';
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

export interface demoFilters {
    steam_id: string;
    map_name: string;
    server_ids: number[];
}

export const apiGetDemos = async (opts: demoFilters) => {
    const demos = await apiCall<DemoFile[]>('/api/demos', 'POST', opts);
    return demos.map((d) => {
        return {
            ...d,
            created_on: parseDateTime(d.created_on as unknown as string)
        };
    });
};
