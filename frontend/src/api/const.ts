export enum Team {
    SPEC,
    RED,
    BLU
}

export enum Weapon {
    ScatterGun
}

export const EventType = {
    UnhandledMsg: 0,
    UnknownMsg: 1,

    // Live player actions
    Say: 10,
    SayTeam: 11,
    Killed: 12,
    KillAssist: 13,
    Suicide: 14,
    ShotFired: 15,
    ShotHit: 16,
    Damage: 17,
    Domination: 18,
    Revenge: 19,
    Pickup: 20,
    EmptyUber: 21,
    MedicDeath: 22,
    MedicDeathEx: 23,
    LostUberAdv: 24,
    ChargeReady: 25,
    ChargeDeployed: 26,
    ChargeEnded: 27,
    Healed: 28,
    Extinguished: 29,
    BuiltObject: 30,
    CarryObject: 31,
    KilledObject: 32,
    DetonatedObject: 33,
    DropObject: 34,
    FirstHealAfterSpawn: 35,
    CaptureBlocked: 36,
    KilledCustom: 37,
    PointCaptured: 48,
    JoinedTeam: 49,
    ChangeClass: 50,
    SpawnedAs: 51,

    // World events not attached to specific players

    WRoundOvertime: 100,
    WRoundStart: 101,
    WRoundWin: 102,
    WRoundLen: 103,
    WTeamScore: 104,
    WTeamFinalScore: 105,
    WGameOver: 106,
    WPaused: 107,
    WResumed: 108,

    // Metadata

    LogStart: 1000,
    LogStop: 1001,
    CVAR: 1002,
    RCON: 1003,
    Connected: 1004,
    Disconnected: 1005,
    Validated: 1006,
    Entered: 1007,

    // Catch-all
    Any: 10000
} as const;

export type EventTypeKey = typeof EventType;

export const eventNames = {
    [EventType.UnhandledMsg]: 'unknown',
    [EventType.UnknownMsg]: 'unknown',
    [EventType.Say]: 'say',
    [EventType.SayTeam]: 'say_team',
    [EventType.Killed]: 'killed',
    [EventType.KillAssist]: 'kill_assist',
    [EventType.Suicide]: 'suicide',
    [EventType.ShotFired]: 'shot_fired',
    [EventType.ShotHit]: 'shot_hit',
    [EventType.Damage]: 'damage',
    [EventType.Domination]: 'domination',
    [EventType.Revenge]: 'revenge',
    [EventType.Pickup]: 'pickup',
    [EventType.EmptyUber]: 'empty_uber',
    [EventType.MedicDeath]: 'medic_death',
    [EventType.MedicDeathEx]: 'medic_death_ex',
    [EventType.LostUberAdv]: 'lost_uber_adv',
    [EventType.ChargeReady]: 'charge_ready',
    [EventType.ChargeDeployed]: 'charge_deployed',
    [EventType.ChargeEnded]: 'charge_ended',
    [EventType.Healed]: 'healed',
    [EventType.Extinguished]: 'extinguished',
    [EventType.BuiltObject]: 'build_object',
    [EventType.CarryObject]: 'carry_object',
    [EventType.KilledObject]: 'killed_object',
    [EventType.DetonatedObject]: 'detonated_object',
    [EventType.DropObject]: 'drop_object',
    [EventType.FirstHealAfterSpawn]: 'first_heal',
    [EventType.CaptureBlocked]: 'capture_blocked',
    [EventType.KilledCustom]: 'killed_custom',
    [EventType.PointCaptured]: 'point_captured',
    [EventType.JoinedTeam]: 'joined_team',
    [EventType.ChangeClass]: 'change_class',
    [EventType.SpawnedAs]: 'spawned_as',
    [EventType.WRoundOvertime]: 'w_round_overtime',
    [EventType.WRoundStart]: 'w_round_start',
    [EventType.WRoundWin]: 'w_round_win',
    [EventType.WRoundLen]: 'w_round_len',
    [EventType.WTeamScore]: 'w_team_score',
    [EventType.WTeamFinalScore]: 'w_team_final_score',
    [EventType.WGameOver]: 'w_game_over',
    [EventType.WPaused]: 'w_paused',
    [EventType.WResumed]: 'w_resumed',
    [EventType.LogStart]: 'log_start',
    [EventType.LogStop]: 'log_stop',
    [EventType.CVAR]: 'cvar',
    [EventType.RCON]: 'rcon',
    [EventType.Connected]: 'connected',
    [EventType.Disconnected]: 'disconnected',
    [EventType.Validated]: 'validated',
    [EventType.Entered]: 'entered',
    [EventType.Any]: 'Any'
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
