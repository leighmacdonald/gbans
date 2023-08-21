import { apiCall } from './common';

export interface DatabaseStats {
    bans: number;
    bans_day: number;
    bans_week: number;
    bans_month: number;
    bans_3month: number;
    bans_6month: number;
    bans_year: number;
    bans_cidr: number;
    appeals_open: number;
    appeals_closed: number;
    filtered_words: number;
    servers_alive: number;
    servers_total: number;
}

export const apiGetStats = async () =>
    await apiCall<DatabaseStats>(`/api/stats`, 'GET');

export interface TeamScores {
    red: number;
    red_time: number;
    blu: number;
    blu_time: number;
}
