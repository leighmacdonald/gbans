import { transformCreatedOnDate } from '../util/time.ts';
import { apiCall, QueryFilter } from './common.ts';

export const Detections = [
    'unknown',
    'silent_aim',
    'aim_snap',
    'too_many_conn',
    'interp',
    'bhop',
    'cmdnum_spike',
    'eye_angles',
    'invalid_user_cmd',
    'oob_cvar',
    'cheat_cvar'
] as const;

export type Detection = (typeof Detections)[number];

export type StacEntry = {
    anticheat_id: number;
    steam_id: string;
    server_id: number;
    server_name: string;
    demo_id: number | null; // Since it's a pointer, it can be null if not set
    demo_name: string;
    demo_tick: number;
    name: string;
    detection: Detection;
    triggered: number;
    summary: string;
    raw_log: string;
    created_on: Date;
    personaname: string;
    avatar: string;
    query: string;
};

export interface AnticheatQuery extends QueryFilter {
    name?: string;
    steam_id?: string;
    server_id?: number;
    summary?: string;
    detection?: Detection;
}

export const apiGetAnticheatLogs = async (query: AnticheatQuery) => {
    return (await apiCall<StacEntry[]>(`/api/anticheat/entries`, 'GET', query)).map(transformCreatedOnDate);
};
