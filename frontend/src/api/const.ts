export type SteamID = bigint;

export enum Team {
    SPEC,
    RED,
    BLU
}

export enum Weapon {
    ScatterGun
}
//
export const EventTypeById = {
    0: 'unhandled',
    1: 'unknown',
    10: 'say',
    11: 'say_team',
    12: 'killed',
    13: 'kill_assist',
    14: 'suicide',
    15: 'shot_fired',
    16: 'shot_hit',
    17: 'damage',
    18: 'domination',
    19: 'revenge',
    20: 'pickup',
    21: 'empty_uber',
    22: 'medic_death',
    23: 'medic_death_ex',
    24: 'lost_uber_adv',
    25: 'charge_ready',
    26: 'charge_deployed',
    27: 'charge_ended',
    28: 'healed',
    29: 'extinguished',
    30: 'build_object',
    31: 'carry_object',
    32: 'killed_object',
    33: 'detonated_object',
    34: 'drop_object',
    35: 'first_heal',
    36: 'capture_blocked',
    37: 'killed_custom',
    48: 'point_captured',
    49: 'joined_team',
    50: 'change_class',
    51: 'spawned_as',
    100: 'w_round_overtime',
    101: 'w_round_start',
    102: 'w_round_win',
    103: 'w_round_len',
    104: 'w_team_score',
    105: 'w_team_final_score',
    106: 'w_game_over',
    107: 'w_paused',
    108: 'w_resumed',
    1000: 'log_start',
    1001: 'log_stop',
    1002: 'cvar',
    1003: 'rcon',
    1004: 'connected',
    1005: 'disconnected',
    1006: 'validated',
    1007: 'entered',
    10000: 'Any'
} as Record<number, string>;
//
// export type EventTypeKey = typeof EventType;

export const EventTypeByName = {
    unhandled: 0,
    unknown: 1,
    say: 10,
    say_team: 11,
    killed: 12,
    kill_assist: 13,
    suicide: 14,
    shot_fired: 15,
    shot_hit: 16,
    damage: 17,
    domination: 18,
    revenge: 19,
    pickup: 20,
    empty_uber: 21,
    medic_death: 22,
    medic_death_ex: 23,
    lost_uber_adv: 24,
    charge_ready: 25,
    charge_deployed: 26,
    charge_ended: 27,
    healed: 28,
    extinguished: 29,
    build_object: 30,
    carry_object: 31,
    killed_object: 32,
    detonated_object: 33,
    drop_object: 34,
    first_heal: 35,
    capture_blocked: 36,
    killed_custom: 37,
    point_captured: 48,
    joined_team: 49,
    change_class: 50,
    spawned_as: 51,
    w_round_overtime: 100,
    w_round_start: 101,
    w_round_win: 102,
    w_round_len: 103,
    w_team_score: 104,
    w_team_final_score: 104,
    w_game_over: 106,
    w_paused: 107,
    w_resumed: 108,
    log_start: 1000,
    log_stop: 1001,
    cvar: 1002,
    rcon: 1003,
    connected: 1004,
    disconnected: 1005,
    validated: 1006,
    entered: 1007,
    Any: 10000
} as Record<string, number>;

export type EventTypeKey = typeof EventTypeByName;

export const eventName = (k: number): string => {
    return EventTypeById[k];
};

export enum PlayerClass {
    Spectator,
    Scout,
    Soldier,
    Pyro,
    Demo,
    Heavy,
    Engineer,
    Medic,
    Sniper,
    Spy,
    Unknown
}

export const PlayerClassNames: Record<PlayerClass, string> = {
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
export enum PickupItem {
    ItemHPSmall,
    ItemHPMedium,
    ItemHPLarge,
    ItemAmmoSmall,
    ItemAmmoMedium,
    ItemAmmoLarge
}
