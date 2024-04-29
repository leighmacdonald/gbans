import { LazyResult } from '../util/table.ts';
import { parseDateTime } from '../util/text.tsx';
import { apiCall, MatchTimes, QueryFilter, transformMatchDates } from './common';
import { PlayerClass, Team } from './const';
import { TeamScores } from './stats';

export interface MatchHealer {
    match_medic_id: number;
    match_player_id: number;
    healing: number;
    charges_uber: number;
    charges_kritz: number;
    charges_vacc: number;
    charges_quickfix: number;
    drops: number;
    near_full_charge_death: number;
    avg_uber_length: number;
    major_adv_lost: number;
    biggest_adv_lost: number;
}

export interface MatchPlayerWeapon {
    weapon_id: number;
    key: string;
    name: string;
    kills: number;
    damage: number;
    shots: number;
    hits: number;
    accuracy: number;
    backstabs: number;
    headshots: number;
    airshots: number;
}

export interface MatchPlayerClass {
    match_player_class_id: number;
    match_player_id: number;
    player_class: PlayerClass;
    kills: number;
    assists: number;
    deaths: number;
    playtime: number;
    dominations: number;
    dominated: number;
    revenges: number;
    damage: number;
    damage_taken: number;
    healing_taken: number;
    captures: number;
    captures_blocked: number;
    building_destroyed: number;
}

export interface MatchPlayerKillstreak {
    match_killstreak_id: number;
    match_player_id: number;
    player_class: PlayerClass;
    killstreak: number;
    duration: number;
}

export interface MatchPlayer {
    match_player_id: number;
    steam_id: string;
    team: Team;
    name: string;
    avatar_hash: string;
    time_start: Date;
    time_end: Date;
    kills: number;
    assists: number;
    deaths: number;
    suicides: number;
    dominations: number;
    dominated: number;
    revenges: number;
    damage: number;
    damage_taken: number;
    healing_taken: number;
    health_packs: number;
    healing_packs: number;
    captures: number;
    captures_blocked: number;
    extinguishes: number;
    building_built: number;
    building_destroyed: number;
    backstabs: number;
    airshots: number;
    headshots: number;
    shots: number;
    hits: number;
    medic_stats: MatchHealer | null;
    classes: MatchPlayerClass[];
    killstreaks: MatchPlayerKillstreak[];
    weapons: MatchPlayerWeapon[];
}

export interface PersonMessages {}

export interface MatchResult extends MatchTimes {
    match_id: string;
    server_id: number;
    title: string;
    map_name: string;
    team_scores: TeamScores;
    players: MatchPlayer[];
    chat: PersonMessages[];
}

export const apiGetMatch = async (match_id: string) => {
    const match = await apiCall<MatchResult>(`/api/log/${match_id}`, 'GET');
    return transformMatchDates(match);
};

export interface MatchesQueryOpts extends QueryFilter<MatchSummary> {
    steam_id?: string;
    server_id?: number;
    map?: string;
    time_start?: Date;
    time_end?: Date;
}

export const apiGetMatches = async (opts: MatchesQueryOpts, abortController: AbortController) => {
    const resp = await apiCall<LazyResult<MatchSummary>, MatchesQueryOpts>(`/api/logs`, 'POST', opts, abortController);
    resp.data = resp.data.map((m) => {
        m.time_start = parseDateTime(m.time_start as unknown as string);
        m.time_end = parseDateTime(m.time_end as unknown as string);
        return m;
    });

    return resp;
};

export interface MatchSummary extends MatchTimes {
    match_id: string;
    server_id: number;
    is_winner: boolean;
    short_name: string;
    title: string;
    map_name: string;
    score_blu: number;
    score_red: number;
    time_start: Date;
    time_end: Date;
}
