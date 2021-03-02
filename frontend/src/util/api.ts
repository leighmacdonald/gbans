import {apiCall, apiError} from "./network";

export interface ChatMessage {
    message: string
    created_on: Date
}

export interface BannedPerson {
    ban: Ban
    person: Person
    history_chat: ChatMessage[]
    history_personaname: string[]
    history_connections: string[]
    history_ip: string[]
}

export interface Ban {
    ban_id?: bigint
    net_id?: bigint
    steam_id?: bigint
    cidr?: string
    author_id?: number
    ban_type?: number
    reason: number
    reason_text?: string
    note?: string
    source: number
    valid_until: string
    created_on: string | Date
    updated_on: string | Date
}

export interface Server {
    server_id: string
    server_name: string
    token: string
    address: string
    port: number
    rcon: string
    reserved_slots: number
    password: string
    token_created_on: string
    created_on: string | Date
    updated_on: string | Date
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
    steamid: string
    communityvisibilitystate: communityVisibilityState
    profilestate: profileState
    personaname: string
    profileurl: string
    avatar: string
    avatarmedium: string
    avatarfull: string
    avatarhash: string
    personastate: number
    realname: string
    primaryclanid: string // ? should be number
    timecreated: number
    personastateflags: number
    loccountrycode: string
    locstatecode: string
    loccityid: number

    // Custom attributes
    steam_id: number
    ip_addr: string
    created_on: string | Date
    updated_on: string | Date
}

export interface IAPIRequestBans {
    offset: number
    limit: number
    sort_desc: boolean
    query: string
    order_by: string
}

export interface IAPIResponseBans {
    bans: BannedPerson[]
    total: number
}

export interface DatabaseStats {
    bans: number
    bans_day: number
    bans_week: number
    bans_month: number
    bans_3month: number
    bans_6month: number
    bans_year: number
    bans_cidr: number
    appeals_open: number
    appeals_closed: number
    filtered_words: number
    servers_alive: number
    servers_total: number
}

export const apiGetStats = async (): Promise<DatabaseStats> => {
    const resp = await apiCall(`/api/v1/stats`, "GET")
    return resp.json as DatabaseStats
}

export interface BanPayload {
    steam_id: string
    duration: string
    ban_type: number
    reason: number
    reason_text: string
    network: string
}

export interface PlayerProfile {
    player: Person
    friends: Person[]
}

export const apiGetBan = async (ban_id: number): Promise<BannedPerson | apiError> => {
    const resp = await apiCall<BannedPerson>(`/api/v1/ban/${ban_id}`, "GET")
    return resp.json
}

export const apiCreateBan = async (p: BanPayload): Promise<Ban | apiError> => {
    const resp = await apiCall<Ban, BanPayload>(`/api/v1/ban`, "POST", p);
    return resp.json;
}

export const apiGetProfile = async (query: string): Promise<PlayerProfile | apiError> => {
    const resp = await apiCall<PlayerProfile>(`/api/v1/profile?query=${query}`, "GET");
    return resp.json;
}

export const apiGetCurrentProfile = async (): Promise<Person | apiError> => {
    const resp = await apiCall<Person>(`/api/v1/current_profile`, "GET");
    return resp.json;
}

export const handleOnLogin = () => {
    const r = `${window.location.protocol}//${window.location.hostname}/auth/callback?return_url=${window.location.pathname}`
    const oid = "https://steamcommunity.com/openid/login" +
        "?openid.ns=" + encodeURIComponent("http://specs.openid.net/auth/2.0") +
        "&openid.mode=checkid_setup" +
        "&openid.return_to=" + encodeURIComponent(r) +
        `&openid.realm=` + encodeURIComponent(`${window.location.protocol}//${window.location.hostname}`) +
        "&openid.ns.sreg=" + encodeURIComponent("http://openid.net/extensions/sreg/1.1") +
        "&openid.claimed_id=" + encodeURIComponent("http://specs.openid.net/auth/2.0/identifier_select") +
        "&openid.identity=" + encodeURIComponent("http://specs.openid.net/auth/2.0/identifier_select")
    window.open(oid, "_self")
}

export const handleOnLogout = () => {
    localStorage.removeItem("token");
}