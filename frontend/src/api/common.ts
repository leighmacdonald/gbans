import { AppError, ErrorCode } from '../error.tsx';
import { readAccessToken } from '../util/auth/readAccessToken.ts';
import { parseDateTime } from '../util/text.tsx';
import { emptyOrNullString } from '../util/types';
import { AppealState, ASNBanRecord, CIDRBanRecord, GroupBanRecord, SteamBanRecord } from './bans';
import { MatchResult } from './match';

export enum PermissionLevel {
    Banned = 0,
    Guest = 1,
    User = 10,
    Reserved = 15,
    Editor = 25,
    Moderator = 50,
    Admin = 100
}
export const PermissionLevelCollection = [
    PermissionLevel.Banned,
    PermissionLevel.Guest,
    PermissionLevel.User,
    PermissionLevel.Reserved,
    PermissionLevel.Editor,
    PermissionLevel.Moderator,
    PermissionLevel.Admin
];

export const permissionLevelString = (level: PermissionLevel) => {
    switch (level) {
        case PermissionLevel.Admin:
            return 'Admin';
        case PermissionLevel.Editor:
            return 'Editor';
        case PermissionLevel.Banned:
            return 'Banned';
        case PermissionLevel.User:
            return 'User';
        case PermissionLevel.Moderator:
            return 'Moderator';
        case PermissionLevel.Reserved:
            return 'VIP';
        case PermissionLevel.Guest:
            return 'Guest';
        default:
            return 'Unknown';
    }
};

export interface DataCount {
    count: number;
}

export class EmptyBody {}

// isRefresh is to track if the token is being used as an auth refresh token. In that
// case its returned instead of the standard access token.
const getAccessToken = async () => {
    try {
        const access = readAccessToken();
        // const refresh = readRefreshToken();
        // if (isTokenExpired(access) && !isTokenExpired(refresh)) {
        //     await refreshToken();
        //     return;
        // }
        return access;
    } catch (e) {
        console.log(`failed to refersh: ${e}`);
        return '';
    }
};

interface errorMessage {
    message: string;
    code?: number;
}

export const apiRootURL = (): string => `${location.protocol}//${location.host}`;

type httpMethods = 'POST' | 'GET' | 'DELETE' | 'PUT';

/**
 * All api requests are handled through this interface.
 *
 * @param url
 * @param method
 * @param body
 * @param abortController
 * @throws AppError
 */
export const apiCall = async <TResponse = EmptyBody | null, TRequestBody = Record<string, unknown> | object>(
    url: string,
    method: httpMethods = 'GET',
    body?: TRequestBody | undefined,
    abortController?: AbortController
): Promise<TResponse> => {
    const headers: Record<string, string> = {
        'Content-Type': 'application/json; charset=UTF-8'
    };
    const requestOptions: RequestInit = {
        mode: 'cors',
        credentials: 'include',
        method: method.toUpperCase()
    };

    const accessToken = await getAccessToken();

    if (!emptyOrNullString(accessToken)) {
        headers['Authorization'] = `Bearer ${accessToken}`;
    }

    requestOptions.headers = headers;

    if (method !== 'GET' && body) {
        requestOptions['body'] = JSON.stringify(body);
    }

    if (abortController != undefined) {
        requestOptions.signal = abortController.signal;
    }

    const response = await fetch(new URL(url, apiRootURL()), requestOptions);

    switch (response.status) {
        case 415:
            throw new AppError(ErrorCode.InvalidMimetype);
        case 424:
            throw new AppError(ErrorCode.DependencyMissing);
        case 401:
            if (accessToken != '') {
                throw new AppError(ErrorCode.LoginRequired);
            }
            break;
        case 403:
            if (accessToken != '') {
                throw new AppError(ErrorCode.PermissionDenied);
            }
    }

    if (!response.ok) {
        let err: errorMessage = { message: 'Error', code: ErrorCode.Unknown };
        try {
            err = (await response.json()) as errorMessage;
        } catch (e) {
            throw new AppError(ErrorCode.Unknown);
        }
        throw new AppError(err.code ?? ErrorCode.Unknown, err.message);
    }

    if (response.status == 204) {
        return null as TResponse;
    }
    return (await response.json()) as TResponse;
};

export class ValidationException extends Error {}

export interface MatchTimes {
    time_start: Date;
    time_end: Date;
}

export interface DateRange {
    date_start: Date;
    date_end: Date;
}

export const transformDateRange = <T>(item: T & DateRange) => {
    item.date_end = parseDateTime(item.date_end as unknown as string);
    item.date_start = parseDateTime(item.date_start as unknown as string);

    return item;
};

export interface TimeStamped {
    created_on: Date;
    updated_on: Date;
    valid_until?: Date;
}

export const transformCreatedOnDate = <T>(item: T & { created_on: Date }) => {
    item.created_on = parseDateTime(item.created_on as unknown as string);
    return item;
};

export const transformTimeStampedDates = <T>(item: T & TimeStamped) => {
    item.created_on = parseDateTime(item.created_on as unknown as string);
    item.updated_on = parseDateTime(item.updated_on as unknown as string);
    if (item.valid_until != undefined) {
        item.valid_until = parseDateTime(item.valid_until as unknown as string);
    }
    return item;
};

export const transformTimeStampedDatesList = <T>(items: (T & TimeStamped)[]) => {
    return items ? items.map(transformTimeStampedDates) : items;
};

export const transformMatchDates = (item: MatchResult) => {
    item.time_start = parseDateTime(item.time_start as unknown as string);
    item.time_end = parseDateTime(item.time_end as unknown as string);
    item.players = item.players.map((t) => {
        t.time_start = parseDateTime(t.time_start as unknown as string);
        t.time_end = parseDateTime(t.time_end as unknown as string);
        return t;
    });
    return item;
};

export interface QueryFilter<T> {
    offset?: number;
    limit?: number;
    desc?: boolean;
    query?: string;
    order_by?: keyof T;
    deleted?: boolean;
    flagged_only?: boolean;
}

export interface BanQueryCommon<T> extends QueryFilter<T> {
    source_id?: string;
    target_id?: string;
    appeal_state?: AppealState;
    deleted?: boolean;
}

export type BanSteamQueryFilter = BanQueryCommon<SteamBanRecord>;

export interface BanCIDRQueryFilter extends BanQueryCommon<CIDRBanRecord> {
    ip?: string;
}

export interface BanGroupQueryFilter extends BanQueryCommon<GroupBanRecord> {
    group_id?: string;
}

export interface BanASNQueryFilter extends BanQueryCommon<ASNBanRecord> {
    as_num?: number;
}

export interface ReportQueryFilter {
    deleted?: boolean;
    source_id?: string;
}

export interface appInfoDetail {
    site_name: string;
    app_version: string;
    link_id: string;
    sentry_dns_web: string;
    asset_url: string;
    patreon_client_id: string;
}
