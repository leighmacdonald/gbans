import { Duration, TimeStamped, transformCreatedOnDate } from '../util/time.ts';
import { apiCall } from './common.ts';
import { UserProfile } from './profile.ts';

export type MapDetail = {
    map_id: number;
    map_name: string;
} & TimeStamped;

export type SpeedrunPointCaptures = {
    speedrun_id: number;
    round_id: number;
    players: SpeedrunParticipant[];
    duration: Duration;
    point_name: string;
};
export type SpeedrunParticipant = {
    person: UserProfile;
    round_id: number;
    steam_id: string;
    kills: number;
    destructions: number;
    duration: Duration;
    persona_name: string;
    avatar_hash: string;
};

export type SpeedrunResult = {
    speedrun_id: number;
    server_id: number;
    rank: number;
    initial_rank: number;
    map_detail: MapDetail;
    point_captures: SpeedrunPointCaptures[];
    players: SpeedrunParticipant[];
    duration: Duration;
    player_count: number;
    bot_count: number;
    created_on: Date;
    category: string;
};

export type SpeedrunMapOverview = {
    speedrun_id: number;
    server_id: number;
    rank: number;
    initial_rank: number;
    map_detail: MapDetail;
    duration: Duration;
    player_count: number;
    bot_count: number;
    created_on: Date;
    category: string;
    total_players: number;
};

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
