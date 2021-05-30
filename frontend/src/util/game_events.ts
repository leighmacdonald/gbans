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

export interface ServerLog {
    log_id: number;
    server_id: number;
    event_type: MsgType;
    payload: unknown;
    source_id: number;
    target_it: number;
    created_on: Date;
}

export enum Team {
    SPEC,
    RED,
    BLU
}

export interface SourcePlayer {
    name: string;
    pid: number;
    sid: number;
    team: Team;
}

export interface TargetPlayer {
    name2: string;
    pid2: number;
    sid2: number;
    team2: Team;
}

export interface SourceEvt extends EmptyEvt {
    source: SourcePlayer;
}

export interface TargetEvt extends SourceEvt {
    target: TargetPlayer;
}

export interface EmptyEvt {
    created_on: number;
}

// type UnhandledMsgEvt = EmptyEvt;
// type EnteredEvt = EmptyEvt;
// type WRoundStartEvt = EmptyEvt;
// type WRoundStartEvt = EmptyEvt;
//
// type WRoundOvertimeEvt = EmptyEvt;
//
// type WPausedEvt = EmptyEvt;
//
// type WResumedEvt = EmptyEvt;

export interface SayEvt extends TargetEvt {
    msg: string;
}
