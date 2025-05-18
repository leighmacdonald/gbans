import { z } from 'zod';
import { schemaQueryFilter } from './query.ts';

export const Team = {
    UNASSIGNED: 0,
    SPEC: 1,
    RED: 2,
    BLU: 3
} as const;

export const TeamEnum = z.nativeEnum(Team);
export type TeamEnum = z.infer<typeof TeamEnum>;

export const PlayerClass = {
    Spectator: 0,
    Scout: 1,
    Soldier: 2,
    Pyro: 3,
    Demo: 4,
    Heavy: 5,
    Engineer: 6,
    Medic: 7,
    Sniper: 8,
    Spy: 9,
    Unknown: 10
} as const;

export const PlayerClassEnum = z.nativeEnum(PlayerClass);
export type PlayerClassEnum = z.infer<typeof PlayerClassEnum>;

export const PlayerClassNames: Record<PlayerClassEnum, string> = {
    [PlayerClass.Spectator]: 'spectator',
    [PlayerClass.Scout]: 'scout',
    [PlayerClass.Soldier]: 'soldier',
    [PlayerClass.Pyro]: 'pyro',
    [PlayerClass.Demo]: 'demo',
    [PlayerClass.Heavy]: 'heavy',
    [PlayerClass.Engineer]: 'engineer',
    [PlayerClass.Medic]: 'medic',
    [PlayerClass.Sniper]: 'sniper',
    [PlayerClass.Spy]: 'spy',
    [PlayerClass.Unknown]: 'unknown'
};

export const schemaMatchHealer = z.object({
    match_medic_id: z.number(),
    match_player_id: z.number(),
    healing: z.number(),
    charges_uber: z.number(),
    charges_kritz: z.number(),
    charges_vacc: z.number(),
    charges_quickfix: z.number(),
    drops: z.number(),
    near_full_charge_death: z.number(),
    avg_uber_length: z.number(),
    major_adv_lost: z.number(),
    biggest_adv_lost: z.number()
});

export type MatchHealer = z.infer<typeof schemaMatchHealer>;

export const schemaMatchPlayerWeapon = z.object({
    weapon_id: z.number(),
    key: z.string(),
    name: z.string(),
    kills: z.number(),
    damage: z.number(),
    shots: z.number(),
    hits: z.number(),
    accuracy: z.number(),
    backstabs: z.number(),
    headshots: z.number(),
    airshots: z.number()
});

export const schemaMatchPlayerClass = z.object({
    match_player_class_id: z.number(),
    match_player_id: z.number(),
    player_class: PlayerClassEnum,
    kills: z.number(),
    assists: z.number(),
    deaths: z.number(),
    playtime: z.number(),
    dominations: z.number(),
    dominated: z.number(),
    revenges: z.number(),
    damage: z.number(),
    damage_taken: z.number(),
    healing_taken: z.number(),
    captures: z.number(),
    captures_blocked: z.number(),
    building_destroyed: z.number()
});
export type MatchPlayerClass = z.infer<typeof schemaMatchPlayerClass>;

export const schemaMatchPlayerKillstreak = z.object({
    match_killstreak_id: z.number(),
    match_player_id: z.number(),
    player_class: PlayerClassEnum,
    killstreak: z.number(),
    duration: z.number()
});

export const schemaMatchPlayer = z.object({
    match_player_id: z.number(),
    steam_id: z.string(),
    team: TeamEnum,
    name: z.string(),
    avatar_hash: z.string(),
    time_start: z.date(),
    time_end: z.date(),
    kills: z.number(),
    assists: z.number(),
    deaths: z.number(),
    suicides: z.number(),
    dominations: z.number(),
    dominated: z.number(),
    revenges: z.number(),
    damage: z.number(),
    damage_taken: z.number(),
    healing_taken: z.number(),
    health_packs: z.number(),
    healing_packs: z.number(),
    captures: z.number(),
    captures_blocked: z.number(),
    extinguishes: z.number(),
    building_built: z.number(),
    building_destroyed: z.number(),
    backstabs: z.number(),
    airshots: z.number(),
    headshots: z.number(),
    shots: z.number(),
    hits: z.number(),
    medic_stats: schemaMatchHealer.optional(),
    classes: z.array(schemaMatchPlayerClass),
    killstreaks: z.array(schemaMatchPlayerKillstreak),
    weapons: z.array(schemaMatchPlayerWeapon)
});

export type MatchPlayer = z.infer<typeof schemaMatchPlayer>;

export const schemaMatchTimes = z.object({
    time_start: z.date(),
    time_end: z.date()
});

export const schemaTeamScores = z.object({
    red: z.number(),
    red_time: z.number(),
    blu: z.number(),
    blu_time: z.number()
});
export type TeamScores = z.infer<typeof schemaTeamScores>;

export const schemaMatchResult = z
    .object({
        match_id: z.string(),
        server_id: z.number(),
        title: z.string(),
        map_name: z.string(),
        team_scores: schemaTeamScores,
        players: z.array(schemaMatchPlayer)
    })
    .merge(schemaMatchTimes);

export type MatchResult = z.infer<typeof schemaMatchResult>;

export const schemaDatabaseStats = z.object({
    bans: z.number(),
    bans_day: z.number(),
    bans_week: z.number(),
    bans_month: z.number(),
    bans_3month: z.number(),
    bans_6month: z.number(),
    bans_year: z.number(),
    bans_cidr: z.number(),
    appeals_open: z.number(),
    appeals_closed: z.number(),
    filtered_words: z.number(),
    servers_alive: z.number(),
    servers_total: z.number()
});

export type DatabaseStats = z.infer<typeof schemaDatabaseStats>;

export const schemaMapUseDetail = z.object({
    map: z.string(),
    playtime: z.number(),
    percent: z.number()
});

export type MapUseDetail = z.infer<typeof schemaMapUseDetail>;

export const schemaWeapon = z.object({
    weapon_id: z.number(),
    key: z.string(),
    name: z.string()
});

export type Weapon = z.infer<typeof schemaWeapon>;

export const schemaBaseWeaponStats = z.object({
    ka: z.number(),
    kills: z.number(),
    assists: z.number(),
    deaths: z.number(),
    kd: z.number(),
    kad: z.number(),
    dpm: z.number(),
    shots: z.number(),
    hits: z.number(),
    accuracy: z.number(),
    airshots: z.number(),
    backstabs: z.number(),
    headshots: z.number(),
    playtime: z.number(),
    dominations: z.number(),
    dominated: z.number(),
    revenges: z.number(),
    damage: z.number(),
    damage_taken: z.number(),
    captures: z.number(),
    captures_blocked: z.number(),
    buildings_destroyed: z.number()
});

export type BaseWeaponStats = z.infer<typeof schemaBaseWeaponStats>;

export const schemaWeaponsOverallResult = z
    .object({
        kills_pct: z.number(),
        damage_pct: z.number(),
        headshots_pct: z.number(),
        airshots_pct: z.number(),
        backstabs_pct: z.number(),
        shots_pct: z.number(),
        hits_pct: z.number()
    })
    .merge(schemaWeapon)
    .merge(schemaBaseWeaponStats);

export type WeaponsOverallResult = z.infer<typeof schemaWeaponsOverallResult>;

export const schemaGamePlayerClass = z.object({
    player_class_id: z.number(),
    class_name: z.string(),
    class_key: z.string()
});
export type GamePlayerClass = z.infer<typeof schemaGamePlayerClass>;

export const schemaMatchPlayerClassStats = z.object({
    kills: z.number(),
    ka: z.number(),
    assists: z.number(),
    deaths: z.number(),
    kd: z.number(),
    kad: z.number(),
    dpm: z.number(),
    playtime: z.number(),
    dominations: z.number(),
    dominated: z.number(),
    revenges: z.number(),
    damage: z.number(),
    damage_taken: z.number(),
    captures: z.number(),
    captures_blocked: z.number(),
    buildings_destroyed: z.number()
});

export const schemaPlayerClassOverallResult = z
    .object({})
    .merge(schemaGamePlayerClass)
    .merge(schemaMatchPlayerClassStats);
export type PlayerClassOverallResult = z.infer<typeof schemaPlayerClassOverallResult>;

export const schemaHealingStats = z.object({
    healing: z.number(),
    drops: z.number(),
    near_full_charge_death: z.number(),
    avg_uber_len: z.number(),
    biggest_adv_lost: z.number(),
    major_adv_lost: z.number(),
    charges_uber: z.number(),
    charges_kritz: z.number(),
    charges_vacc: z.number(),
    charges_quickfix: z.number(),
    hpm: z.number()
});

export const schemaWinSums = z.object({
    wins: z.number(),
    matches: z.number(),
    win_rate: z.number()
});

export const schemaHealingOverallResult = z
    .object({
        rank: z.number(),
        steam_id: z.string(),
        personaname: z.string(),
        avatar_hash: z.string(),
        ka: z.number(),
        assists: z.number(),
        deaths: z.number(),
        kad: z.number(),
        playtime: z.number(),
        dominations: z.number(),
        dominated: z.number(),
        revenges: z.number(),
        damage_taken: z.number(),
        dtm: z.number(),
        extinguishes: z.number(),
        health_packs: z.number()
    })
    .merge(schemaWinSums)
    .merge(schemaHealingStats);

export type HealingOverallResult = z.infer<typeof schemaHealingOverallResult>;

export const schemaPlayerOverallResult = z
    .object({
        buildings: z.number(),
        extinguishes: z.number(),
        health_packs: z.number(),
        shots: z.number(),
        hits: z.number(),
        accuracy: z.number(),
        airshots: z.number(),
        backstabs: z.number(),
        headshots: z.number(),
        healing_taken: z.number()
    })
    .merge(schemaHealingStats)
    .merge(schemaMatchPlayerClassStats)
    .merge(schemaWinSums);

export type PlayerOverallResult = z.infer<typeof schemaPlayerOverallResult>;

export const schemaPlayerWeaponStats = z
    .object({
        steam_id: z.string(),
        personaname: z.string(),
        avatar_hash: z.string(),
        rank: z.number()
    })
    .merge(schemaBaseWeaponStats);

export type PlayerWeaponStats = z.infer<typeof schemaPlayerWeaponStats>;

export const schemaMatchSummary = z
    .object({
        match_id: z.string(),
        server_id: z.number(),
        is_winner: z.boolean(),
        short_name: z.string(),
        title: z.string(),
        map_name: z.string(),
        score_blu: z.number(),
        score_red: z.number(),
        time_start: z.date(),
        time_end: z.date()
    })
    .merge(schemaMatchTimes);

export type MatchSummary = z.infer<typeof schemaMatchSummary>;

export const schemaMatchesQueryOpts = z
    .object({
        steam_id: z.string().optional(),
        server_id: z.number().optional(),
        map: z.string().optional(),
        time_start: z.date().optional(),
        time_end: z.date().optional()
    })
    .merge(schemaQueryFilter);

export type MatchesQueryOpts = z.infer<typeof schemaMatchesQueryOpts>;

export const schemaMedicRow = z
    .object({
        steam_id: z.string(),
        team: TeamEnum,
        name: z.string(),
        avatar_hash: z.string(),
        time_start: z.date(),
        time_end: z.date()
    })
    .merge(schemaMatchHealer);

export type MedicRow = z.infer<typeof schemaMedicRow>;
