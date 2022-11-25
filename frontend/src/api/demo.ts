import { apiCall } from './common';
import { parseDateTime } from '../util/text';

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
}

export interface demoFilters {
    steamId: string;
    mapName: string;
    serverId: number;
}
export const apiGetDemos = async (opts: demoFilters) => {
    const resp = await apiCall<DemoFile[]>('/api/demos', 'POST', opts);
    if (resp.result) {
        resp.result = resp.result.map((row) => {
            return {
                ...row,
                created_on: parseDateTime(row.created_on as unknown as string)
            };
        });
    }
    return resp;
};
