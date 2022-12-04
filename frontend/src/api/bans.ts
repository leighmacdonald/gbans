import { apiCall, QueryFilter, TimeStamped } from './common';
import {
    communityVisibilityState,
    Person,
    profileState,
    UserProfile
} from './profile';
import { UserMessage } from './report';
import SteamID from 'steamid';
import { parseDateTime } from '../util/text';

export enum AppealState {
    Open,
    Denied,
    Accepted,
    Reduced,
    NoAppeal
}
export const AppealStateCollection = [
    AppealState.Open,
    AppealState.Denied,
    AppealState.Accepted,
    AppealState.Reduced,
    AppealState.NoAppeal
];

export const appealStateString = (as: AppealState): string => {
    switch (as) {
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
    BotHost = 12
}

export const ip2int = (ip: string): number =>
    ip
        .split('.')
        .reduce((ipInt, octet) => (ipInt << 8) + parseInt(octet, 10), 0) >>> 0;

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

export const Durations = [
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
    [BanReason.BotHost]: 'Bot Host'
};

export const banReasonsList = [
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
    BanReason.BotHost
];

export enum BanType {
    Unknown = -1,
    OK = 0,
    NoComm = 1,
    Banned = 2
}

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

export interface BannedPerson {
    ban: IAPIBanRecord;
    person: Person;
}

export interface BanBase extends TimeStamped {
    valid_until: Date;
    reason: BanReason;
    reason_text: string;
    source_id: SteamID;
    target_id: SteamID;
    deleted: boolean;
    unban_reason_text: string;
    note: string;
    origin: Origin;
    appeal_state: AppealState;
}

export interface IAPIBanRecord extends BanBase {
    ban_id: number;
    report_id: number;
    ban_type: BanType;
}

export interface SimplePerson {
    steam_id: SteamID;
    persona_name: string;
    avatar: string;
    avatar_full: string;
}

export interface AppealOverview extends IAPIBanRecord {
    source_steam_id: SteamID;
    source_persona_name: string;
    source_avatar: string;
    source_avatar_full: string;

    target_steam_id: SteamID;
    target_persona_name: string;
    target_avatar: string;
    target_avatar_full: string;
}

export interface IAPIBanGroupRecord extends BanBase {
    ban_group_id: number;
    group_id: SteamID;
    group_name: string;
}

export interface IAPIBanCIDRRecord extends BanBase {
    net_id: number;
    cidr: string;
}

export interface IAPIBanASNRecord extends BanBase {
    ban_asn_id: bigint;
    as_num: number;
}

export type IAPIResponseBans = BannedPerson[];

export interface IAPIBanRecordProfile extends IAPIBanRecord {
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

export interface BansQueryFilter extends QueryFilter<IAPIBanRecordProfile> {
    steam_id?: SteamID;
}

export interface UnbanPayload {
    unban_reason_text: string;
}

export interface BanBasePayload {
    target_id: string;
    duration: string;
    ban_type: BanType;
    reason: number;
    reason_text: string;
    note: string;
}

export interface BanPayloadSteam extends BanBasePayload {
    report_id?: number;
}

export interface BanPayloadCIDR extends BanBasePayload {
    cidr: string;
}

export interface BanPayloadASN extends BanBasePayload {
    as_num: number;
}

export const apiGetBansSteam = async (opts?: BansQueryFilter) => {
    const resp = await apiCall<IAPIResponseBans, BansQueryFilter>(
        `/api/bans/steam`,
        'POST',
        opts ?? {}
    );
    return (resp.result ?? [])
        .map((b): IAPIBanRecordProfile => {
            return {
                source_id: b.ban.source_id,
                avatar: b.person.avatar,
                avatarfull: b.person.avatarfull,
                avatarmedium: b.person.avatarmedium,
                ban_id: b.ban.ban_id,
                ban_type: b.ban.ban_type,
                communityvisibilitystate: b.person.communityvisibilitystate,
                ip_addr: b.person.ip_addr,
                loccountrycode: b.person.loccountrycode,
                note: b.ban.note,
                personaname: b.person.personaname,
                personastate: b.person.personastate,
                personastateflags: b.person.personastateflags,
                profilestate: b.person.profilestate,
                profileurl: b.person.profileurl,
                realname: b.person.realname,
                reason: b.ban.reason,
                reason_text: b.ban.reason_text,
                origin: b.ban.origin,
                target_id: b.ban.target_id,
                timecreated: b.person.timecreated,
                deleted: b.ban.deleted,
                report_id: b.ban.report_id,
                unban_reason_text: b.ban.unban_reason_text,
                appeal_state: b.ban.appeal_state,
                created_on: b.ban.created_on,
                updated_on: b.ban.updated_on,
                valid_until: b.ban.valid_until
            };
        })
        .map(applyDateTime);
};

export function applyDateTime<T>(row: T & TimeStamped) {
    const record = {
        ...row,
        created_on: parseDateTime(row.created_on as unknown as string),
        updated_on: parseDateTime(row.updated_on as unknown as string)
    };
    if (record?.valid_until) {
        record.valid_until = parseDateTime(
            record.valid_until as unknown as string
        );
    }
    return record;
}

export const apiGetBanSteam = async (ban_id: number, deleted = false) => {
    const resp = await apiCall<BannedPerson>(
        `/api/bans/steam/${ban_id}?deleted=${deleted}`,
        'GET'
    );
    if (resp.result?.ban && resp.result?.person) {
        resp.result.ban = applyDateTime(resp.result?.ban);
        resp.result.person = applyDateTime(resp.result?.person);
    }
    return resp;
};

export const apiGetAppeals = async (opts?: BansQueryFilter) => {
    const resp = await apiCall<AppealOverview[]>(
        `/api/appeals`,
        'POST',
        opts ?? {}
    );
    if (resp.status && resp.result) {
        resp.result = resp.result.map((r) => applyDateTime(r));
    }
    return resp;
};

export const apiCreateBanSteam = async (p: BanPayloadSteam) =>
    await apiCall<IAPIBanRecord, BanPayloadSteam>(
        `/api/bans/steam/create`,
        'POST',
        p
    );

export const apiCreateBanCIDR = async (p: BanPayloadCIDR) =>
    await apiCall<IAPIBanCIDRRecord, BanPayloadCIDR>(
        `/api/bans/cidr/create`,
        'POST',
        p
    );

export const apiCreateBanASN = async (p: BanPayloadASN) =>
    await apiCall<IAPIBanASNRecord, BanPayloadASN>(
        `/api/bans/asn/create`,
        'POST',
        p
    );

export const apiCreateBanGroup = async (p: BanBasePayload) =>
    await apiCall<IAPIBanGroupRecord, BanBasePayload>(
        `/api/bans/group/create`,
        'POST',
        p
    );

export const apiDeleteBan = async (ban_id: number, unban_reason_text: string) =>
    await apiCall<null, UnbanPayload>(`/api/bans/steam/${ban_id}`, 'DELETE', {
        unban_reason_text
    });

export interface AuthorMessage {
    message: UserMessage;
    author: UserProfile;
}

export const apiGetBanMessages = async (ban_id: number) => {
    const resp = await apiCall<AuthorMessage[]>(
        `/api/bans/${ban_id}/messages`,
        'GET'
    );
    if (resp.result) {
        resp.result = resp.result.map((r) => {
            return {
                message: applyDateTime(r.message),
                author: applyDateTime(r.author)
            };
        });
    }
    return resp;
};

export interface CreateBanMessage {
    message: string;
}

export const apiCreateBanMessage = async (ban_id: number, message: string) =>
    await apiCall<UserMessage, CreateBanMessage>(
        `/api/bans/${ban_id}/messages`,
        'POST',
        { message }
    );

export const apiUpdateBanMessage = async (
    ban_message_id: number,
    message: string
) =>
    await apiCall(`/api/bans/message/${ban_message_id}`, 'POST', {
        body_md: message
    });

export const apiDeleteBanMessage = async (ban_message_id: number) =>
    await apiCall(`/api/bans/message/${ban_message_id}`, 'DELETE', {});

export const apiGetBansCIDR = async (opts: QueryFilter<IAPIBanCIDRRecord>) => {
    const resp = await apiCall<IAPIBanCIDRRecord[]>(
        '/api/bans/cidr',
        'POST',
        opts
    );
    if (resp.status && resp.result) {
        resp.result = resp.result.map((record) => applyDateTime(record));
    }
    return resp;
};

export const apiGetBansASN = async (opts: QueryFilter<IAPIBanASNRecord>) => {
    const resp = await apiCall<IAPIBanASNRecord[]>(
        '/api/bans/asn',
        'POST',
        opts
    );
    if (resp.status && resp.result) {
        resp.result = resp.result.map((record) => applyDateTime(record));
    }
    return resp;
};

export const apiGetBansGroups = async (
    opts: QueryFilter<IAPIBanGroupRecord>
) => {
    const resp = await apiCall<IAPIBanGroupRecord[]>(
        '/api/bans/group',
        'POST',
        opts
    );
    if (resp.status && resp.result) {
        resp.result = resp.result.map((record) => applyDateTime(record));
    }
    return resp;
};

export const apiDeleteCIDRBan = async (
    cidr_id: number,
    unban_reason_text: string
) =>
    await apiCall<null, UnbanPayload>(`/api/bans/cidr/${cidr_id}`, 'DELETE', {
        unban_reason_text
    });

export const apiDeleteASNBan = async (
    as_num: number,
    unban_reason_text: string
) =>
    await apiCall<null, UnbanPayload>(`/api/bans/asn/${as_num}`, 'DELETE', {
        unban_reason_text
    });

export const apiDeleteGroupBan = async (
    ban_group_id: number,
    unban_reason_text: string
) =>
    await apiCall<null, UnbanPayload>(
        `/api/bans/group/${ban_group_id}`,
        'DELETE',
        {
            unban_reason_text
        }
    );

export const apiSetBanAppealState = async (
    ban_id: number,
    appeal_state: AppealState
) =>
    await apiCall(`/api/bans/steam/${ban_id}/status`, 'POST', {
        appeal_state
    });
