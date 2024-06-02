import { apiCall, transformCreatedOnDate } from './common';

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

export const apiGetDemos = async () => {
    const resp = await apiCall<DemoFile[]>('/api/demos', 'POST', undefined);
    return resp.map(transformCreatedOnDate);
};

export const apiGetDemoCleanup = async () => await apiCall('/api/demos/cleanup');
