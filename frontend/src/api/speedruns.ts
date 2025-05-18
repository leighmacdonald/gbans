import { SpeedrunMapOverview, SpeedrunResult } from '../schema/speedrun.ts';
import { transformCreatedOnDate } from '../util/time.ts';
import { apiCall } from './common.ts';

export const getSpeedrunsTopOverall = async (count: number = 5) => {
    const results = await apiCall<Record<string, SpeedrunResult[]>>(`/api/speedruns/overall/top?count=${count}`, 'GET');
    for (const key of Object.keys(results)) {
        results[key] = results[key].map(transformCreatedOnDate);
    }
    return results;
};

export const getSpeedrunsTopMap = async (map_name: string) => {
    return (await apiCall<SpeedrunMapOverview[]>(`/api/speedruns/map?map_name=${map_name}`, 'GET')).map(
        transformCreatedOnDate
    );
};

export const getSpeedrunsRecent = async (count: number = 5) => {
    return (await apiCall<SpeedrunMapOverview[]>(`/api/speedruns/overall/recent?count=${count}`, 'GET')).map(
        transformCreatedOnDate
    );
};

export const getSpeedrun = async (speedrun_id: number) => {
    const r = transformCreatedOnDate(await apiCall<SpeedrunResult>(`/api/speedruns/byid/${speedrun_id}`, 'GET'));
    r.players = r.players.sort((a, b) => {
        return a.duration > b.duration ? 1 : -1;
    });
    r.point_captures.map((p) => {
        p.players = p.players.sort((a, b) => {
            return a.duration > b.duration ? 1 : -1;
        });
    });
    return r;
};
