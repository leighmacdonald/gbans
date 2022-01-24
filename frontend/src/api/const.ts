export enum Team {
    SPEC,
    RED,
    BLU
}

export enum Weapon {
    ScatterGun
}

export enum MsgType {
    UnhandledMsg = 0,
    UnknownMsg = 1,

    // Live player actions
    Say = 10,
    SayTeam = 11,
    Killed = 12,
    KillAssist = 13,
    Suicide = 14,
    ShotFired = 15,
    ShotHit = 16,
    Damage = 17,
    Domination = 18,
    Revenge = 19,
    Pickup = 20,
    EmptyUber = 21,
    MedicDeath = 22,
    MedicDeathEx = 23,
    LostUberAdv = 24,
    ChargeReady = 25,
    ChargeDeployed = 26,
    ChargeEnded = 27,
    Healed = 28,
    Extinguished = 29,
    BuiltObject = 30,
    CarryObject = 31,
    KilledObject = 32,
    DetonatedObject = 33,
    DropObject = 34,
    FirstHealAfterSpawn = 35,
    CaptureBlocked = 36,
    KilledCustom = 37,
    PointCaptured = 48,
    JoinedTeam = 49,
    ChangeClass = 50,
    SpawnedAs = 51,

    // World events not attached to specific players

    WRoundOvertime = 100,
    WRoundStart = 101,
    WRoundWin = 102,
    WRoundLen = 103,
    WTeamScore = 104,
    WTeamFinalScore = 105,
    WGameOver = 106,
    WPaused = 107,
    WResumed = 108,

    // Metadata

    LogStart = 1000,
    LogStop = 1001,
    CVAR = 1002,
    RCON = 1003,
    Connected = 1004,
    Disconnected = 1005,
    Validated = 1006,
    Entered = 1007,

    // Catch-all
    Any = 10000
}

export const eventNames: Record<MsgType, string> = {
    [MsgType.UnhandledMsg]: 'unknown',
    [MsgType.UnknownMsg]: 'unknown',
    [MsgType.Say]: 'say',
    [MsgType.SayTeam]: 'say_team',
    [MsgType.Killed]: 'killed',
    [MsgType.KillAssist]: 'kill_assist',
    [MsgType.Suicide]: 'suicide',
    [MsgType.ShotFired]: 'shot_fired',
    [MsgType.ShotHit]: 'shot_hit',
    [MsgType.Damage]: 'damage',
    [MsgType.Domination]: 'domination',
    [MsgType.Revenge]: 'revenge',
    [MsgType.Pickup]: 'pickup',
    [MsgType.EmptyUber]: 'empty_uber',
    [MsgType.MedicDeath]: 'medic_death',
    [MsgType.MedicDeathEx]: 'medic_death_ex',
    [MsgType.LostUberAdv]: 'lost_uber_adv',
    [MsgType.ChargeReady]: 'charge_ready',
    [MsgType.ChargeDeployed]: 'charge_deployed',
    [MsgType.ChargeEnded]: 'charge_ended',
    [MsgType.Healed]: 'healed',
    [MsgType.Extinguished]: 'extinguished',
    [MsgType.BuiltObject]: 'build_object',
    [MsgType.CarryObject]: 'carry_object',
    [MsgType.KilledObject]: 'killed_object',
    [MsgType.DetonatedObject]: 'detonated_object',
    [MsgType.DropObject]: 'drop_object',
    [MsgType.FirstHealAfterSpawn]: 'first_heal',
    [MsgType.CaptureBlocked]: 'capture_blocked',
    [MsgType.KilledCustom]: 'killed_custom',
    [MsgType.PointCaptured]: 'point_captured',
    [MsgType.JoinedTeam]: 'joined_team',
    [MsgType.ChangeClass]: 'change_class',
    [MsgType.SpawnedAs]: 'spawned_as',
    [MsgType.WRoundOvertime]: 'w_round_overtime',
    [MsgType.WRoundStart]: 'w_round_start',
    [MsgType.WRoundWin]: 'w_round_win',
    [MsgType.WRoundLen]: 'w_round_len',
    [MsgType.WTeamScore]: 'w_team_score',
    [MsgType.WTeamFinalScore]: 'w_team_final_score',
    [MsgType.WGameOver]: 'w_game_over',
    [MsgType.WPaused]: 'w_paused',
    [MsgType.WResumed]: 'w_resumed',
    [MsgType.LogStart]: 'log_start',
    [MsgType.LogStop]: 'log_stop',
    [MsgType.CVAR]: 'cvar',
    [MsgType.RCON]: 'rcon',
    [MsgType.Connected]: 'connected',
    [MsgType.Disconnected]: 'disconnected',
    [MsgType.Validated]: 'validated',
    [MsgType.Entered]: 'entered',
    [MsgType.Any]: 'Any'
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
