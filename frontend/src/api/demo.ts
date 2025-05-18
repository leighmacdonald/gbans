import { DemoFile } from '../schema/demo.ts';
import { transformCreatedOnDate } from '../util/time.ts';
import { apiCall } from './common';

export const apiGetDemos = async () => {
    const resp = await apiCall<DemoFile[]>('/api/demos', 'POST', undefined);
    return resp.map(transformCreatedOnDate);
};

export const apiGetDemoCleanup = async () => await apiCall('/api/demos/cleanup');
