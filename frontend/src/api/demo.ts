import { apiCall } from './common';

export interface DemoFile {
    demo_id: number;
    server_id: number;
    title: string;
    created_on: string;
    size: number;
    downloads: number;
    map_name: string;
    archive: boolean;
}

export interface demoFilters {
    steam_id: string;
    map_name: string;
    server_id: number;
}
export const apiGetDemos = (opts: demoFilters) =>
    apiCall<DemoFile>('/api/demos', 'POST', opts);
