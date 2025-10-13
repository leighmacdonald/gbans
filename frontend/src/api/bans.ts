import {
    AppealQueryFilter,
    AppealState,
    AppealStateEnum,
    BanPayload,
    BanType,
    BanTypeEnum,
    BodyMDMessage,
    sbBanRecord,
    BanRecord,
    UnbanPayload,
    UpdateBanPayload
} from '../schema/bans.ts';
import { TimeStampedWithValidUntil } from '../schema/chrono.ts';
import { BanQueryOpts } from '../schema/query.ts';
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

export const apiGetBans = async (opts: BanQueryOpts, abortController?: AbortController) => {
    const resp = await apiCall<BanRecord[], BanQueryOpts>(`/api/bans`, 'GET', opts, abortController);
    return resp.map(transformTimeStampedDates);
};

export const apiGetBansSteamBySteamID = async (steam_id: string, abortController?: AbortController) => {
    const resp = await apiCall<BanRecord[], BanQueryOpts>(
        `/api/bans/all/${steam_id}`,
        'GET',
        undefined,
        abortController
    );
    return resp.map(transformTimeStampedDates);
};

export const apiGetBanBySteam = async (steamID: string, abortController?: AbortController) =>
    transformTimeStampedDates(
        await apiCall<BanRecord>(`/api/bans/steamid/${steamID}`, 'GET', undefined, abortController)
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
    const resp = await apiCall<BanRecord>(`/api/bans/${ban_id}?deleted=${deleted}`, 'GET', undefined, abortController);

    return resp ? transformTimeStampedDates(resp) : undefined;
};

export const apiGetAppeals = async (opts: AppealQueryFilter, abortController?: AbortController) => {
    const appeals = await apiCall<BanRecord[], AppealQueryFilter>(`/api/appeals`, 'POST', opts, abortController);
    return appeals.map(applyDateTime);
};

export const apiCreateBan = async (p: BanPayload) =>
    transformTimeStampedDates(await apiCall<BanRecord, BanPayload>(`/api/bans`, 'POST', p));

export const apiUpdateBanSteam = async (ban_id: number, payload: UpdateBanPayload) =>
    transformTimeStampedDates(await apiCall<BanRecord, UpdateBanPayload>(`/api/bans/${ban_id}`, 'POST', payload));

export const apiDeleteBan = async (ban_id: number, unban_reason_text: string) =>
    await apiCall<null, UnbanPayload>(`/api/bans/${ban_id}`, 'DELETE', {
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

export const apiSetBanAppealState = async (ban_id: number, appeal_state: AppealStateEnum) =>
    await apiCall(`/api/bans/${ban_id}/status`, 'POST', {
        appeal_state
    });

export const apiGetSourceBans = async (steam_id: string) => {
    const resp = await apiCall<sbBanRecord[]>(`/api/sourcebans/${steam_id}`, 'GET');
    return resp.map(transformCreatedOnDate);
};
