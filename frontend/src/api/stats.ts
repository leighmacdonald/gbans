import { apiCall, QueryFilter } from './common';
import { Person } from './profile';
import SteamID from 'steamid';
import { parseDateTime } from '../util/text';

export interface CommonStats {
    kills: number;
    assists: number;
    damage: number;
    healing: number;
    shots: number;
    hits: number;
    suicides: number;
    extinguishes: number;
    point_captures: number;
    point_defends: number;
    medic_dropped_uber: number;
    object_built: number;
    object_destroyed: number;
    messages: number;
    messages_team: number;
    pickup_ammo_large: number;
    pickup_ammo_medium: number;
    pickup_ammo_small: number;
    pickup_hp_large: number;
    pickup_hp_medium: number;
    pickup_hp_small: number;
    spawn_scout: number;
    spawn_soldier: number;
    spawn_pyro: number;
    spawn_demo: number;
    spawn_heavy: number;
    spawn_engineer: number;
    spawn_medic: number;
    spawn_spy: number;
    spawn_sniper: number;
    dominations: number;
    revenges: number;
    playtime: number;
}

export interface PlayerStats extends CommonStats {
    deaths: number;
    games: number;
    wins: number;
    losses: number;
    damage_taken: number;
    dominated: number;
}

export type ServerStats = CommonStats;

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

export interface MatchPlayerSum {
    MatchPlayerSumID: number;
    SteamId: SteamID;
    Team: number;
    TimeStart?: string;
    TimeEnd?: string;
    Kills: number;
    Assists: number;
    KDRatio: number;
    KADRatio: number;
    Deaths: number;
    Dominations: number;
    Dominated: number;
    Revenges: number;
    Damage: number;
    DamageTaken: number;
    Healing: number;
    HealingTaken: number;
    HealthPacks: number;
    BackStabs: number;
    HeadShots: number;
    Airshots: number;
    Captures: number;
    Shots: number;
    Hits: number;
    Extinguishes: number;
    BuildingBuilt: number;
    BuildingDestroyed: number;
    Classes: number;
}

export interface MatchMedicSum {
    MatchMedicId: number;
    MatchId: number;
    SteamId: SteamID;
    Healing: number;
    Charges: { [key: number]: number };
    Drops: number;
    AvgTimeToBuild: number;
    AvgTimeBeforeUse: number;
    NearFullChargeDeath: number;
    AvgUberLength: number;
    DeathAfterCharge: number;
    MajorAdvLost: number;
    BiggestAdvLost: number;
    HealTargets: { [key: number]: number };
}

export const ratio = (a: number, b: number): number => {
    return a / b;
};

export interface MatchTeamSum {
    MatchTeamId: number;
    MatchId: number;
    Team: number;
    Kills: number;
    Damage: number;
    Charges: number;
    Drops: number;
    Caps: number;
    MidFights: number;
}

export interface TeamScores {
    Red: number;
    Blu: number;
}

export interface MatchRoundSum {
    Length: number;
    Score: TeamScores;
    KillsBlu: number;
    KillsRed: number;
    UbersBlu: number;
    UbersRed: number;
    DamageBlu: number;
    DamageRed: number;
    MidFight: number;
}

export interface MatchClassSums {
    Scout: number;
    Soldier: number;
    Pyro: number;
    Demoman: number;
    Heavy: number;
    Engineer: number;
    Medic: number;
    Sniper: number;
    Spy: number;
}

export interface MatchSummary {
    match_id: number;
    server_id: number;
    map_name: string;
    created_on: Date;
    player_count: number;
    kills: number;
    assists: number;
    damage: number;
    healing: number;
    airshots: number;
}

export interface BaseMatch {
    MatchID: number;
    ServerId: number;
    Title: string;
    MapName: string;
    CreatedOn: Date;
}

export interface Match extends BaseMatch {
    PlayerSums: MatchPlayerSum[];
    MedicSums: MatchMedicSum[];
    TeamSums: MatchTeamSum[];
    Rounds: MatchRoundSum[];
    ClassKills: { [key: number]: MatchClassSums };
    ClassKillsAssists: { [key: number]: MatchClassSums };
    ClassDeaths: { [key: number]: MatchClassSums };
    Players: Person[];
}

export const apiGetMatch = async (match_id: number) =>
    await apiCall<Match>(`/api/log/${match_id}`, 'GET');

export interface MatchesQueryOpts extends QueryFilter<MatchSummary> {
    steam_id?: SteamID;
    server_id?: number;
    map?: string;
    time_start?: Date;
    time_end?: Date;
}

export const apiGetMatches = async (opts: MatchesQueryOpts) =>
    await apiCall<MatchSummary[], MatchesQueryOpts>(`/api/logs`, 'POST', opts);

export interface LocalTF2StatSnapshot {
    stat_id: number;
    players: number;
    capacity_full: number;
    capacity_empty: number;
    capacity_partial: number;
    map_types: Record<string, number>;
    servers: Record<string, number>;
    created_on: Date;
}

export interface GlobalTF2StatSnapshot extends LocalTF2StatSnapshot {
    bots: number;
    secure: number;
    servers_community: number;
    servers_total: number;
}

export enum StatDuration {
    Live,
    Hourly,
    Daily,
    Weekly,
    Yearly
}

export const statDurationString = (duration: StatDuration): string => {
    switch (duration) {
        case StatDuration.Live:
            return 'Live';
        case StatDuration.Hourly:
            return 'Hourly';
        case StatDuration.Daily:
            return 'Daily';
        case StatDuration.Weekly:
            return 'Weekly';
        case StatDuration.Yearly:
            return 'Yearly';
    }
};

export const StatDurations = [
    StatDuration.Live,
    StatDuration.Hourly,
    StatDuration.Daily,
    StatDuration.Weekly,
    StatDuration.Yearly
];

export enum StatSource {
    Local,
    Global
}

export const apiGetTF2Stats = async (
    source: StatSource,
    duration: StatDuration
) =>
    await apiCall<GlobalTF2StatSnapshot[] | LocalTF2StatSnapshot[]>(
        `/api/server_stats?source=${
            source === StatSource.Local ? 'local' : 'global'
        }&duration=${duration}`,
        'GET'
    ).then((resp) => {
        if (!resp.result) {
            return resp;
        }
        resp.result = resp.result?.map((r) => {
            return {
                ...r,
                created_on: parseDateTime(r.created_on as unknown as string)
            };
        });
        return resp;
    });
