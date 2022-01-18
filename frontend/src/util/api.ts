import { log } from './errors';

export enum PermissionLevel {
    Guest = 1,
    Banned = 2,
    Authenticated = 10,
    Moderator = 50,
    Admin = 100
}

export interface apiResponse<T> {
    status: boolean;
    resp: Response;
    json: T | apiError;
}

export interface apiError {
    error?: string;
}

const apiCall = async <TResponse, TRequestBody = Record<string, unknown>>(
    url: string,
    method: string,
    body?: TRequestBody
): Promise<apiResponse<TResponse>> => {
    const headers: Record<string, string> = {
        'Content-Type': 'application/json; charset=UTF-8'
    };
    const opts: RequestInit = {
        mode: 'cors',
        credentials: 'include',
        method: method.toUpperCase()
    };
    const token = localStorage.getItem('token');
    if (token != '') {
        headers['Authorization'] = `Bearer ${token}`;
    }
    if (method === 'POST' && body) {
        opts['body'] = JSON.stringify(body);
    }
    opts.headers = headers;
    const resp = await fetch(url, opts);
    if (resp.status === 403 && token != '') {
        log('invalid token');
    }
    if (!resp.status) {
        throw apiErr('Invalid response code', resp);
    }
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const json = ((await resp.json()) as TResponse as any).data;
    if (json?.error && json.error !== '') {
        throw apiErr(`Error received: ${json.error}`, resp);
    }
    return { json: json, resp: resp, status: resp.ok };
};

class ApiException extends Error {
    public resp: Response;

    constructor(msg: string, response: Response) {
        super(msg);
        this.resp = response;
    }
}

const apiErr = (msg: string, resp: Response): ApiException => {
    return new ApiException(msg, resp);
};

export enum PayloadType {
    okType,
    errType,
    authType,
    authOKType,
    logType,
    logQueryOpts,
    logQueryResults
}

// Used for setting filtering / query options for realtime log event streams
export interface LogQueryOpts {
    log_types: MsgType[];
    limit: number;
    order_desc: boolean;
    query: string;
    source_id: string;
    target_id: string;
    servers: number[];
}

export const encode = (t: PayloadType, o: unknown): WebSocketPayload => {
    return {
        payload_type: t,
        data: o
    };
};

export interface WebSocketPayload<TRecord = unknown> {
    payload_type: PayloadType;
    data: TRecord;
}

export interface WebSocketAuthResp {
    status: boolean;
    message: string;
}

export interface ChatMessage {
    message: string;
    created_on: Date;
}

export interface BannedPerson {
    ban: Ban;
    person: Person;
    history_chat: ChatMessage[];
    history_personaname: string[];
    history_connections: string[];
    history_ip: string[];
}

export interface Ban {
    ban_id: number;
    net_id: number;
    steam_id: number;
    cidr: string;
    author_id: number;
    ban_type: number;
    reason: number;
    reason_text: string;
    note: string;
    source: number;
    valid_until: Date;
    created_on: Date;
    updated_on: Date;
}

export interface PlayerInfo {
    steam_id: number;
    name: string;
    user_id: number;
    connected_time: number;
}

export interface Server {
    server_id: number;
    server_name: string;
    server_name_long: string;
    address: string;
    port: number;
    password_protected: boolean;
    vac: boolean;
    region: string;
    cc: string;
    latitude: number;
    longitude: number;
    current_map: string;
    tags: string[];
    default_map: string;
    reserved_slots: number;
    players_max: number;
    players: PlayerInfo[];
    created_on: Date;
    updated_on: Date;
}

export enum profileState {
    Incomplete = 0,
    Setup = 1
}

export enum communityVisibilityState {
    Private = 1,
    FriendOnly = 2,
    Public = 3
}

export interface Person {
    // PlayerSummaries shape
    steamid: string;
    communityvisibilitystate: communityVisibilityState;
    profilestate: profileState;
    personaname: string;
    profileurl: string;
    avatar: string;
    avatarmedium: string;
    avatarfull: string;
    avatarhash: string;
    personastate: number;
    realname: string;
    primaryclanid: string; // ? should be number
    timecreated: number;
    personastateflags: number;
    loccountrycode: string;
    locstatecode: string;
    loccityid: number;

    // Custom attributes
    steam_id: string;
    ip_addr: string;
    created_on: Date;
    updated_on: Date;
}

export interface QueryFilterProps {
    offset: number;
    limit: number;
    sort_desc: boolean;
    query: string;
    order_by: string;
}

export type IAPIResponseBans = BannedPerson[];

export interface IAPIBanRecord {
    ban_id: number;
    net_id: number;
    steam_id: string;
    cidr: string;
    author_id: number;
    ban_type: number;
    reason: number;
    reason_text: string;
    note: string;
    source: number;
    valid_until: Date;
    created_on: Date;
    updated_on: Date;

    steamid: string;
    communityvisibilitystate: communityVisibilityState;
    profilestate: profileState;
    personaname: string;
    profileurl: string;
    avatar: string;
    avatarmedium: string;
    avatarfull: string;
    personastate: number;
    realname: string;
    timecreated: number;
    personastateflags: number;
    loccountrycode: string;

    // Custom attributes
    ip_addr: string;
}

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

export const apiGetStats = async (): Promise<DatabaseStats> => {
    const resp = await apiCall<DatabaseStats>(`/api/stats`, 'GET');
    return resp.json as DatabaseStats;
};

export interface BanPayload {
    steam_id: string;
    duration: string;
    ban_type: number;
    reason: number;
    reason_text: string;
    network: string;
}

export interface PlayerProfile {
    player: Person;
    friends: Person[];
}

export const apiGetBans = async (): Promise<IAPIBanRecord[] | apiError> => {
    const resp = await apiCall<IAPIResponseBans, QueryFilterProps>(
        `/api/bans`,
        'POST'
    );
    return ((resp.json as IAPIResponseBans) ?? []).map((b): IAPIBanRecord => {
        return {
            author_id: b.ban.author_id,
            avatar: b.person.avatar,
            avatarfull: b.person.avatarfull,
            avatarmedium: b.person.avatarmedium,
            ban_id: b.ban.ban_id,
            ban_type: b.ban.ban_type,
            cidr: b.ban.cidr,
            communityvisibilitystate: b.person.communityvisibilitystate,
            created_on: b.ban.created_on,
            ip_addr: b.person.ip_addr,
            loccountrycode: b.person.loccountrycode,
            net_id: b.ban.net_id,
            note: b.ban.note,
            personaname: b.person.personaname,
            personastate: b.person.personastate,
            personastateflags: b.person.personastateflags,
            profilestate: b.person.profilestate,
            profileurl: b.person.profileurl,
            realname: b.person.realname,
            reason: b.ban.reason,
            reason_text: b.ban.reason_text,
            source: b.ban.source,
            steam_id: b.person.steam_id,
            steamid: b.person.steamid,
            timecreated: b.person.timecreated,
            updated_on: b.ban.updated_on,
            valid_until: b.ban.valid_until
        };
    });
};

export const apiGetBan = async (
    ban_id: number
): Promise<BannedPerson | apiError> => {
    const resp = await apiCall<BannedPerson>(`/api/ban/${ban_id}`, 'GET');
    return resp.json;
};

export const apiCreateBan = async (p: BanPayload): Promise<Ban | apiError> => {
    const resp = await apiCall<Ban, BanPayload>(`/api/ban`, 'POST', p);
    return resp.json;
};

export const apiGetProfile = async (
    query: string
): Promise<PlayerProfile | apiError> => {
    const resp = await apiCall<PlayerProfile>(
        `/api/profile?query=${query}`,
        'GET'
    );
    return resp.json;
};

export const apiGetCurrentProfile = async (): Promise<
    PlayerProfile | apiError
> => {
    const resp = await apiCall<PlayerProfile>(`/api/current_profile`, 'GET');
    return resp.json;
};

export const apiGetServers = async (): Promise<Server[] | apiError> => {
    const resp = await apiCall<Server[]>(`/api/servers`, 'GET');
    return resp.json;
};

export const apiGetPeople = async (): Promise<Person[] | apiError> => {
    const resp = await apiCall<Person[]>(`/api/players`, 'GET');
    return resp.json;
};

export const handleOnLogin = (): void => {
    let returnUrl = window.location.hostname;
    if (
        (window.location.protocol === 'https:' &&
            window.location.port !== '443') ||
        (window.location.protocol === 'http:' &&
            window.location.port !== '80') ||
        (window.location.port != '80' && window.location.port != '443')
    ) {
        returnUrl = `${returnUrl}:${window.location.port}`;
    }
    const r = `${window.location.protocol}//${returnUrl}/auth/callback?return_url=${window.location.pathname}`;
    const oid =
        'https://steamcommunity.com/openid/login' +
        '?openid.ns=' +
        encodeURIComponent('http://specs.openid.net/auth/2.0') +
        '&openid.mode=checkid_setup' +
        '&openid.return_to=' +
        encodeURIComponent(r) +
        `&openid.realm=` +
        encodeURIComponent(
            `${window.location.protocol}//${window.location.hostname}`
        ) +
        '&openid.ns.sreg=' +
        encodeURIComponent('http://openid.net/extensions/sreg/1.1') +
        '&openid.claimed_id=' +
        encodeURIComponent(
            'http://specs.openid.net/auth/2.0/identifier_select'
        ) +
        '&openid.identity=' +
        encodeURIComponent(
            'http://specs.openid.net/auth/2.0/identifier_select'
        );
    window.open(oid, '_self');
};

export const handleOnLogout = (): void => {
    localStorage.removeItem('token');
    location.reload();
};

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

// Helper
export const StringIsNumber = (value: unknown) => !isNaN(Number(value));

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

export interface GlobalStats extends CommonStats {
    unique_players: number;
}

export type ServerStats = CommonStats;
export type MapStats = CommonStats;

export interface Pos {
    x: number;
    y: number;
    z: number;
}

export enum Team {
    SPEC,
    RED,
    BLU
}

export enum Weapon {
    ScatterGun
}

export interface ServerEvent {
    log_id: number;
    server: Server;
    event_type: MsgType;
    source?: Person;
    target?: Person;
    player_class: PlayerClass;
    weapon: Weapon;
    damage: number;
    healing: number;
    item: PickupItem;
    attacker_pos?: Pos;
    victim_pos?: Pos;
    assister_pos?: Pos;
    extra?: string;
    team: Team;
    created_on: Date;
    meta_data?: Record<string, unknown>;
}

export interface Appeal {
    appeal_id: number;
}
