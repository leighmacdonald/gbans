export interface IBanState {
    ban_id: bigint
    steam_id: bigint
    author_id: number
    ban_type: number
    reason: number
    reason_text: string
    note: string
    source: number
    valid_until: number
    created_on: number
    updated_on: number
    personaname: string
    avatar: string
    avatarfull: string
    avatarmedium: string
}

export interface IAPIRequestBans {
    offset: number
    limit: number
    sort_desc: boolean
    query: string
    order_by: string
}

export interface IAPIResponseBans {
    bans: IBanState[]
    total: number
}