import { Duration, TimeStamped, transformCreatedOnDate } from '../util/time.ts';
import { apiCall } from './common.ts';

export type MapDetail = {
    map_id: number;
    map_name: string;
} & TimeStamped;

export type SpeedrunPointCaptures = {
    speedrun_id: number;
    round_id: number;
    players: SpeedrunPointCaptures[];
    duration: Duration;
    point_name: string;
};
export type SpeedrunParticipant = {
    round_id: number;
    steam_id: string;
    duration: Duration;
};

export type SpeedrunResult = {
    speedrun_id: number;
    server_id: number;
    rank: number;
    map_detail: MapDetail;
    point_captures: SpeedrunPointCaptures[];
    players: SpeedrunParticipant[];
    duration: Duration;
    player_count: number;
    bot_count: number;
    created_on: Date;
    category: string;
};

export const getSpeedrunsTopOverall = async (count: number = 5) => {
    const results = await apiCall<Record<string, SpeedrunResult[]>>(`/api/speedruns/overall/top?count=${count}`, 'GET');
    for (const key of Object.keys(results)) {
        results[key] = results[key].map(transformCreatedOnDate);
    }
    return results;
};

export const getSpeedrunsForMap = async (map_name: string) => {
    return await apiCall<SpeedrunResult[]>(`/api/speeduns/map?map_name=${map_name}`, 'GET');
};
