import { Config } from '../schema/config.ts';
import { apiCall } from './common.ts';

export const apiSaveSettings = async (settings: Config) => {
    return await apiCall(`/api/config`, 'PUT', settings);
};

export const apiGetSettings = async () => {
    return await apiCall<Config>('/api/config', 'GET');
};
