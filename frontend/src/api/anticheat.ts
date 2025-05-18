import { AnticheatQuery, StacEntry } from '../schema/anticheat.ts';
import { transformCreatedOnDate } from '../util/time.ts';
import { apiCall } from './common.ts';

export const apiGetAnticheatLogs = async (query: AnticheatQuery) => {
    return (await apiCall<StacEntry[]>(`/api/anticheat/entries`, 'GET', query)).map(transformCreatedOnDate);
};
