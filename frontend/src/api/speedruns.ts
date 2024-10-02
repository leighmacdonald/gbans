import { apiCall } from './common.ts';

export type SpeedrunResult = {
    map_name: string;
    category: string;
};

export const getSpeedrunsOverall = async () => {
    return await apiCall<SpeedrunResult[]>(`/api/speeduns/overall`, 'GET');
};

export const getSpeedrunsForMap = async (map_name: string) => {
    return await apiCall<SpeedrunResult[]>(`/api/speeduns/map?map_name=${map_name}`, 'GET');
};
