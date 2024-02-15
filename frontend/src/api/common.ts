import { logErr } from '../util/errors';
import { parseDateTime } from '../util/text';
import { emptyOrNullString } from '../util/types';
import {
    isTokenExpired,
    readAccessToken,
    readRefreshToken,
    refreshToken
} from './auth';
import {
    AppealState,
    ASNBanRecord,
    CIDRBanRecord,
    GroupBanRecord,
    SteamBanRecord
} from './bans';
import { MatchResult } from './match';
import { ReportStatus, ReportWithAuthor } from './report';

export enum PermissionLevel {
    Banned = 0,
    Guest = 1,
    User = 10,
    Reserved = 15,
    Editor = 25,
    Moderator = 50,
    Admin = 100
}

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
const getAccessToken = async (isRefresh: boolean) => {
    if (
        isTokenExpired(readAccessToken()) &&
        !isTokenExpired(readRefreshToken()) &&
        !isRefresh
    ) {
        await refreshToken();
    }

    return isRefresh ? readRefreshToken() : readAccessToken();
};

export enum ErrorCode {
    InvalidMimetype,
    DependencyMissing,
    PermissionDenied,
    Unknown,
    LoginRequired
}

export class APIError extends Error {
    public code: ErrorCode;

    constructor(code: ErrorCode, message?: string, options?: ErrorOptions) {
        if (emptyOrNullString(message)) {
            switch (code) {
                case ErrorCode.InvalidMimetype:
                    message = 'Forbidden file format (mimetype)';
                    break;
                case ErrorCode.DependencyMissing:
                    message = 'Dependency missing, cannot continue';
                    break;
                case ErrorCode.PermissionDenied:
                    message = 'Permission Denied';
                    break;
                case ErrorCode.LoginRequired:
                    message = 'Please Login';
                    break;
                default:
                    message = 'Unhandled Error';
            }
        }
        super(message, options);
        this.code = code;
    }
}

interface errorMessage {
    message: string;
    code?: number;
}

const apiRootURL = (): string => {
    if (import.meta.env.DEV) {
        return 'http://gbans.localhost:6006';
    } else {
        return `${location.protocol}//${location.host}`;
    }
};
/**
 * All api requests are handled through this interface.
 *
 * @param url
 * @param method
 * @param body
 * @param isRefresh
 * @param abortController
 * @throws APIError
 */
export const apiCall = async <
    TResponse = EmptyBody,
    TRequestBody = Record<string, unknown> | object
>(
    url: string,
    method: string = 'GET',
    body?: TRequestBody | undefined,
    abortController?: AbortController,
    isRefresh?: boolean
): Promise<TResponse> => {
    const headers: Record<string, string> = {
        'Content-Type': 'application/json; charset=UTF-8'
    };
    const requestOptions: RequestInit = {
        mode: import.meta.env.DEV ? 'no-cors' : 'same-origin',
        credentials: 'include',
        method: method.toUpperCase()
    };
    let accessToken = '';
    try {
        accessToken = await getAccessToken(isRefresh ?? false);
    } catch (e) {
        logErr(e);
    }
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
            throw new APIError(ErrorCode.InvalidMimetype);
        case 424:
            throw new APIError(ErrorCode.DependencyMissing);
        case 401:
            if (accessToken != '') {
                throw new APIError(ErrorCode.LoginRequired);
            }
            break;
        case 403:
            if (accessToken != '') {
                throw new APIError(ErrorCode.PermissionDenied);
            }
    }

    if (!response.ok) {
        let err: errorMessage = { message: 'Error', code: ErrorCode.Unknown };
        try {
            err = (await response.json()) as errorMessage;
        } catch (e) {
            throw new APIError(ErrorCode.Unknown);
        }
        throw new APIError(err.code ?? ErrorCode.Unknown, err.message);
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

export const transformTimeStampedDatesList = <T>(
    items: (T & TimeStamped)[]
) => {
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

export interface ReportQueryFilter extends QueryFilter<ReportWithAuthor> {
    report_status?: ReportStatus;
    source_id?: string;
    target_id?: string;
}
