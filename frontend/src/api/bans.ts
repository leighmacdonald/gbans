import { LazyResult } from '../util/table.ts';
import { parseDateTime } from '../util/text.tsx';
import {
    apiCall,
    BanASNQueryFilter,
    BanCIDRQueryFilter,
    BanGroupQueryFilter,
    BanSteamQueryFilter,
    QueryFilter,
    TimeStamped,
    transformTimeStampedDates,
    transformTimeStampedDatesList
} from './common';
import { BanAppealMessage } from './report';

export enum AppealState {
    Any = -1,
    Open,
    Denied,
    Accepted,
    Reduced,
    NoAppeal
}

export const AppealStateCollection = [
    AppealState.Any,
    AppealState.Open,
    AppealState.Denied,
    AppealState.Accepted,
    AppealState.Reduced,
    AppealState.NoAppeal
];

export const appealStateString = (as: AppealState): string => {
    switch (as) {
        case AppealState.Any:
            return 'Any';
        case AppealState.Open:
            return 'Open';
        case AppealState.Denied:
            return 'Denied';
        case AppealState.Accepted:
            return 'Accepted';
        case AppealState.Reduced:
            return 'Reduced';
        default:
            return 'No Appeal';
    }
};

export enum Origin {
    System = 0,
    Bot = 1,
    Web = 2,
    InGame = 3
}

export enum BanReason {
    Custom = 1,
    External = 2,
    Cheating = 3,
    Racism = 4,
    Harassment = 5,
    Exploiting = 6,
    WarningsExceeded = 7,
    Spam = 8,
    Language = 9,
    Profile = 10,
    ItemDescriptions = 11,
    BotHost = 12,
    Evading = 13
}

export enum Duration {
    dur15m = '15m',
    dur6h = '6h',
    dur12h = '12h',
    dur24h = '24h',
    dur48h = '48h',
    dur72h = '72h',
    dur1w = '1w',
    dur2w = '2w',
    dur1M = '1M',
    dur6M = '6M',
    dur1y = '1y',
    durInf = '0',
    durCustom = 'custom'
}

export const DurationCollection = [
    Duration.dur15m,
    Duration.dur6h,
    Duration.dur12h,
    Duration.dur24h,
    Duration.dur48h,
    Duration.dur72h,
    Duration.dur1w,
    Duration.dur2w,
    Duration.dur1M,
    Duration.dur6M,
    Duration.dur1y,
    Duration.durInf,
    Duration.durCustom
];

export const BanReasons: Record<BanReason, string> = {
    [BanReason.Custom]: 'Custom',
    [BanReason.External]: '3rd party',
    [BanReason.Cheating]: 'Cheating',
    [BanReason.Racism]: 'Racism',
    [BanReason.Harassment]: 'Personal Harassment',
    [BanReason.Exploiting]: 'Exploiting',
    [BanReason.WarningsExceeded]: 'Warnings Exceeded',
    [BanReason.Spam]: 'Spam',
    [BanReason.Language]: 'Language',
    [BanReason.Profile]: 'Inappropriate Steam Profile',
    [BanReason.ItemDescriptions]: 'Item Name/Descriptions',
    [BanReason.BotHost]: 'Bot Host',
    [BanReason.Evading]: 'Evading'
};

export const banReasonsCollection = [
    BanReason.Cheating,
    BanReason.Racism,
    BanReason.Harassment,
    BanReason.Exploiting,
    BanReason.WarningsExceeded,
    BanReason.Spam,
    BanReason.Language,
    BanReason.Profile,
    BanReason.ItemDescriptions,
    BanReason.External,
    BanReason.Custom,
    BanReason.BotHost,
    BanReason.Evading
];

export enum BanType {
    Unknown = -1,
    OK = 0,
    NoComm = 1,
    Banned = 2
}

export const BanTypeCollection = [BanType.OK, BanType.NoComm, BanType.Banned];

export const banTypeString = (bt: BanType) => {
    switch (bt) {
        case BanType.Banned:
            return 'Banned';
        case BanType.NoComm:
            return 'Muted';
        default:
            return 'Not Banned';
    }
};

export interface BanBase extends TimeStamped {
    valid_until: Date;
    reason: BanReason;
    ban_type: BanType;
    reason_text: string;
    source_id: string;
    target_id: string;
    deleted: boolean;
    unban_reason_text: string;
    note: string;
    origin: Origin;
    appeal_state: AppealState;
    source_personaname: string;
    source_avatarhash: string;
    target_personaname: string;
    target_avatarhash: string;
}

export type SteamBanRecord = {
    ban_id: number;
    report_id: number;
    ban_type: BanType;
    include_friends: boolean;
    evade_ok: boolean;
} & BanBase;

export interface GroupBanRecord extends BanBase {
    ban_group_id: number;
    group_id: string;
    group_name: string;
}

export interface CIDRBanRecord extends BanBase {
    net_id: number;
    cidr: string;
}

export interface ASNBanRecord extends BanBase {
    ban_asn_id: number;
    as_num: number;
}

export interface UnbanPayload {
    unban_reason_text: string;
}

export interface BanBasePayload {
    target_id: string;
    duration: string;
    valid_until?: Date;
    note: string;
}

interface BanReasonPayload {
    reason: BanReason;
    reason_text: string;
}

export interface BanPayloadSteam extends BanBasePayload, BanReasonPayload {
    report_id?: number;
    include_friends: boolean;
    evade_ok: boolean;
    ban_type: BanType;
}

export interface BanPayloadCIDR extends BanBasePayload, BanReasonPayload {
    cidr: string;
}

export interface BanPayloadASN extends BanBasePayload, BanReasonPayload {
    as_num: number;
}

export interface BanPayloadGroup extends BanBasePayload {
    group_id: string;
}

export const apiGetBansSteam = async (opts: BanSteamQueryFilter, abortController?: AbortController) => {
    const resp = await apiCall<LazyResult<SteamBanRecord>, BanSteamQueryFilter>(
        `/api/bans/steam`,
        'POST',
        opts,
        abortController
    );
    resp.data = resp.data.map(applyDateTime);

    return resp;
};

export function applyDateTime<T>(row: T & TimeStamped) {
    const record = {
        ...row,
        created_on: parseDateTime(row.created_on as unknown as string),
        updated_on: parseDateTime(row.updated_on as unknown as string)
    };
    if (record?.valid_until) {
        record.valid_until = parseDateTime(record.valid_until as unknown as string);
    }
    return record;
}

export const apiGetBanSteam = async (ban_id: number, deleted = false, abortController?: AbortController) => {
    const resp = await apiCall<SteamBanRecord>(
        `/api/bans/steam/${ban_id}?deleted=${deleted}`,
        'GET',
        undefined,
        abortController
    );

    return resp ? transformTimeStampedDates(resp) : undefined;
};

export interface AppealQueryFilter extends QueryFilter<SteamBanRecord> {
    source_id?: string;
    target_id?: string;
    appeal_state: AppealState;
}

export const apiGetAppeals = async (opts: AppealQueryFilter, abortController?: AbortController) => {
    const appeals = await apiCall<LazyResult<SteamBanRecord>, AppealQueryFilter>(
        `/api/appeals`,
        'POST',
        opts,
        abortController
    );
    appeals.data = appeals.data.map(applyDateTime);
    return appeals;
};

export const apiCreateBanSteam = async (p: BanPayloadSteam) =>
    transformTimeStampedDates(await apiCall<SteamBanRecord, BanPayloadSteam>(`/api/bans/steam/create`, 'POST', p));

interface UpdateBanPayload {
    reason: BanReason;
    reason_text: string;
    note: string;
    valid_until?: Date;
}

export const apiUpdateBanSteam = async (
    ban_id: number,
    payload: UpdateBanPayload & {
        include_friends: boolean;
        evade_ok: boolean;
        ban_type: BanType;
    }
) =>
    transformTimeStampedDates(
        await apiCall<SteamBanRecord, UpdateBanPayload>(`/api/bans/steam/${ban_id}`, 'POST', payload)
    );

export const apiCreateBanCIDR = async (payload: BanPayloadCIDR) =>
    transformTimeStampedDates(await apiCall<CIDRBanRecord, BanPayloadCIDR>(`/api/bans/cidr/create`, 'POST', payload));

export const apiUpdateBanCIDR = async (
    ban_id: number,
    payload: UpdateBanPayload & {
        cidr: string;
        target_id: string;
    }
) =>
    transformTimeStampedDates(
        await apiCall<CIDRBanRecord, UpdateBanPayload>(`/api/bans/cidr/${ban_id}`, 'POST', payload)
    );
export const apiCreateBanASN = async (payload: BanPayloadASN) =>
    transformTimeStampedDates(await apiCall<ASNBanRecord, BanPayloadASN>(`/api/bans/asn/create`, 'POST', payload));

interface UpdateBanASNPayload {
    target_id: string;
    reason: BanReason;
    as_num: number;
    reason_text: string;
    note: string;
    valid_until?: Date;
}

export const apiUpdateBanASN = async (asn: number, payload: UpdateBanASNPayload) =>
    await apiCall<ASNBanRecord, UpdateBanASNPayload>(`/api/bans/asn/${asn}`, 'POST', payload);

export const apiCreateBanGroup = async (payload: BanPayloadGroup) =>
    transformTimeStampedDates(
        await apiCall<GroupBanRecord, BanPayloadGroup>(`/api/bans/group/create`, 'POST', payload)
    );

interface UpdateBanGroupPayload {
    target_id: string;
    note: string;
    valid_until?: Date;
}

export const apiUpdateBanGroup = async (ban_group_id: number, payload: UpdateBanGroupPayload) =>
    await apiCall<GroupBanRecord, UpdateBanGroupPayload>(`/api/bans/group/${ban_group_id}`, 'POST', payload);
export const apiDeleteBan = async (ban_id: number, unban_reason_text: string) =>
    await apiCall<null, UnbanPayload>(`/api/bans/steam/${ban_id}`, 'DELETE', {
        unban_reason_text
    });

export const apiGetBanMessages = async (ban_id: number) => {
    const resp = await apiCall<BanAppealMessage[]>(`/api/bans/${ban_id}/messages`, 'GET');

    return transformTimeStampedDatesList(resp);
};

export interface CreateBanMessage {
    message: string;
}

export const apiCreateBanMessage = async (ban_id: number, message: string) => {
    const resp = await apiCall<BanAppealMessage, CreateBanMessage>(`/api/bans/${ban_id}/messages`, 'POST', { message });

    return transformTimeStampedDates(resp);
};

export const apiUpdateBanMessage = async (ban_message_id: number, message: string) =>
    transformTimeStampedDates(
        await apiCall<BanAppealMessage>(`/api/bans/message/${ban_message_id}`, 'POST', {
            body_md: message
        })
    );

export const apiDeleteBanMessage = async (ban_message_id: number) =>
    await apiCall(`/api/bans/message/${ban_message_id}`, 'DELETE', {});

export const apiGetBansCIDR = async (opts: BanCIDRQueryFilter, abortController?: AbortController) => {
    const resp = await apiCall<LazyResult<CIDRBanRecord>>('/api/bans/cidr', 'POST', opts, abortController);

    resp.data = resp.data.map((record) => applyDateTime(record));
    return resp;
};

export const apiGetBansASN = async (opts: BanASNQueryFilter, abortController?: AbortController) => {
    const resp = await apiCall<LazyResult<ASNBanRecord>>('/api/bans/asn', 'POST', opts, abortController);
    resp.data = resp.data.map(applyDateTime);
    return resp;
};

export const apiGetBansGroups = async (opts: BanGroupQueryFilter, abortController?: AbortController) => {
    const resp = await apiCall<LazyResult<GroupBanRecord>>('/api/bans/group', 'POST', opts, abortController);

    resp.data = resp.data.map(applyDateTime);
    return resp;
};

export const apiDeleteCIDRBan = async (cidr_id: number, unban_reason_text: string) =>
    await apiCall<null, UnbanPayload>(`/api/bans/cidr/${cidr_id}`, 'DELETE', {
        unban_reason_text
    });

export const apiDeleteASNBan = async (as_num: number, unban_reason_text: string) =>
    await apiCall<null, UnbanPayload>(`/api/bans/asn/${as_num}`, 'DELETE', {
        unban_reason_text
    });

export const apiDeleteGroupBan = async (ban_group_id: number, unban_reason_text: string) =>
    await apiCall<null, UnbanPayload>(`/api/bans/group/${ban_group_id}`, 'DELETE', {
        unban_reason_text
    });

export const apiSetBanAppealState = async (ban_id: number, appeal_state: AppealState) =>
    await apiCall(`/api/bans/steam/${ban_id}/status`, 'POST', {
        appeal_state
    });

export interface sbBanRecord {
    ban_id: number;
    site_id: number;
    site_name: string;
    persona_name: string;
    steam_id: string;
    reason: string;
    duration: number;
    permanent: string;
    created_on: string;
}

export const apiGetSourceBans = async (steam_id: string) =>
    await apiCall<sbBanRecord[]>(`/api/sourcebans/${steam_id}`, 'GET');
