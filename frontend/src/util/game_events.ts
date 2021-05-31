import { Person, Server } from './api';

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

// Helper
export const StringIsNumber = (value: unknown) => !isNaN(Number(value));

export const MsgTypeValues = Object.keys(MsgType)
    .filter(StringIsNumber)
    .map((key) => MsgType[key as any]);

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

export interface Pos {
    x: number;
    y: number;
    z: number;
}

export interface LogEvent {
    event_type: MsgType;
    event: Record<string, string | number | Pos>;
    server: Server;
    player1?: Person;
    player2?: Person;
    assister?: Person;
    raw_event: string;
    created_on: string;
}

export interface ServerLog {
    log_id: number;
    server_id: number;
    event_type: MsgType;
    payload: unknown;
    source_id: string;
    target_id: string;
    created_on: string;
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
    created_on: string;
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

export interface DamageEvt extends TargetEvt {
    damage: number;
    real_damage?: number;
    weapon: string;
    healing?: number;
}
