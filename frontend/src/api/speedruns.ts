import { apiCall } from './common.ts';

type speedrunResult = {
    map_name: string;
};

export const getSpeedrunsOverall = async () => {
    return await apiCall<speedrunResult[]>(`/api/speeduns/overall`, 'GET');
};

export const getSpeedrunsForMap = async (map_name: string) => {
    return await apiCall<speedrunResult[]>(`/api/speeduns/map?map_name=${map_name}`, 'GET');
};
