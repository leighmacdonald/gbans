import {
    AppealQueryFilter,
    AppealState,
    AppealStateEnum,
    ASNBanRecord,
    BanPayloadASN,
    BanPayloadCIDR,
    BanPayloadGroup,
    BanPayloadSteam,
    BanType,
    BanTypeEnum,
    BodyMDMessage,
    CIDRBanRecord,
    GroupBanRecord,
    sbBanRecord,
    SteamBanRecord,
    UnbanPayload,
    UpdateBanASNPayload,
    UpdateBanGroupPayload,
    UpdateBanPayload,
    UpdateBanSteamPayload
} from '../schema/bans.ts';
import { TimeStampedWithValidUntil } from '../schema/chrono.ts';
import { BanASNQueryFilter, BanCIDRQueryFilter, BanGroupQueryFilter, BanSteamQueryFilter } from '../schema/query.ts';
import { BanAppealMessage } from '../schema/report.ts';
import {
    parseDateTime,
    transformCreatedOnDate,
    transformTimeStampedDates,
    transformTimeStampedDatesList
} from '../util/time.ts';
import { apiCall } from './common';

export const appealStateString = (as: AppealStateEnum): string => {
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

export const banTypeString = (bt: BanTypeEnum) => {
    switch (bt) {
        case BanType.Banned:
            return 'Banned';
        case BanType.NoComm:
            return 'Muted';
        default:
            return 'Not Banned';
    }
};

export const apiGetBansSteam = async (opts: BanSteamQueryFilter, abortController?: AbortController) => {
    const resp = await apiCall<SteamBanRecord[], BanSteamQueryFilter>(`/api/bans/steam`, 'GET', opts, abortController);
    return resp.map(transformTimeStampedDates);
};

export const apiGetBansSteamBySteamID = async (steam_id: string, abortController?: AbortController) => {
    const resp = await apiCall<SteamBanRecord[], BanSteamQueryFilter>(
        `/api/bans/steam_all/${steam_id}`,
        'GET',
        undefined,
        abortController
    );
    return resp.map(transformTimeStampedDates);
};

export const apiGetBanBySteam = async (steamID: string, abortController?: AbortController) =>
    transformTimeStampedDates(
        await apiCall<SteamBanRecord>(`/api/bans/steamid/${steamID}`, 'GET', undefined, abortController)
    );

export function applyDateTime<T>(row: T & TimeStampedWithValidUntil) {
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

export const apiGetAppeals = async (opts: AppealQueryFilter, abortController?: AbortController) => {
    const appeals = await apiCall<SteamBanRecord[], AppealQueryFilter>(`/api/appeals`, 'POST', opts, abortController);
    return appeals.map(applyDateTime);
};

export const apiCreateBanSteam = async (p: BanPayloadSteam) =>
    transformTimeStampedDates(await apiCall<SteamBanRecord, BanPayloadSteam>(`/api/bans/steam/create`, 'POST', p));

export const apiUpdateBanSteam = async (ban_id: number, payload: UpdateBanSteamPayload) =>
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

export const apiUpdateBanASN = async (asn: number, payload: UpdateBanASNPayload) =>
    await apiCall<ASNBanRecord, UpdateBanASNPayload>(`/api/bans/asn/${asn}`, 'POST', payload);

export const apiCreateBanGroup = async (payload: BanPayloadGroup) =>
    transformTimeStampedDates(
        await apiCall<GroupBanRecord, BanPayloadGroup>(`/api/bans/group/create`, 'POST', payload)
    );

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

export const apiCreateBanMessage = async (ban_id: number, body_md: string) => {
    const resp = await apiCall<BanAppealMessage, BodyMDMessage>(`/api/bans/${ban_id}/messages`, 'POST', { body_md });

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
    const resp = await apiCall<CIDRBanRecord[]>('/api/bans/cidr', 'GET', opts, abortController);
    return resp.map(transformTimeStampedDates);
};

export const apiGetBansASN = async (opts: BanASNQueryFilter, abortController?: AbortController) => {
    const resp = await apiCall<ASNBanRecord[]>('/api/bans/asn', 'GET', opts, abortController);
    return resp.map(transformTimeStampedDates);
};

export const apiGetBansGroups = async (opts: BanGroupQueryFilter, abortController?: AbortController) => {
    const resp = await apiCall<GroupBanRecord[]>('/api/bans/group', 'GET', opts, abortController);
    return resp.map(transformTimeStampedDates);
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

export const apiSetBanAppealState = async (ban_id: number, appeal_state: AppealStateEnum) =>
    await apiCall(`/api/bans/steam/${ban_id}/status`, 'POST', {
        appeal_state
    });

export const apiGetSourceBans = async (steam_id: string) => {
    const resp = await apiCall<sbBanRecord[]>(`/api/sourcebans/${steam_id}`, 'GET');
    return resp.map(transformCreatedOnDate);
};
