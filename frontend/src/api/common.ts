import { format, parseISO } from 'date-fns';
import { parseDateTime } from '../util/text';
import { emptyOrNullString } from '../util/types';
import {
    isTokenExpired,
    readAccessToken,
    readRefreshToken,
    refreshToken
} from './auth';
import { MatchResult } from './match';
import { ReportStatus } from './report';

export enum PermissionLevel {
    Banned = 0,
    Guest = 1,
    User = 10,
    Editor = 25,
    Moderator = 50,
    Admin = 100
}

export interface DataCount {
    count: number;
}

export class EmptyBody {}

// isRefresh is to track if the token is being used as a auth refresh token. In that
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
    Unknown
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
                default:
                    message = 'Unhandled Error';
            }
        }
        super(message, options);
        this.code = code;
    }
}

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
    TResponse,
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
        mode: 'same-origin',
        credentials: 'include',
        method: method.toUpperCase()
    };

    const accessToken = await getAccessToken(isRefresh ?? false);

    if (accessToken != '') {
        headers['Authorization'] = `Bearer ${accessToken}`;
    }

    requestOptions.headers = headers;

    if (method !== 'GET' && body) {
        requestOptions['body'] = JSON.stringify(body);
    }

    if (abortController != undefined) {
        requestOptions.signal = abortController.signal;
    }

    const response = await fetch(
        new URL(url, `${location.protocol}//${location.host}`),
        requestOptions
    );
    switch (response.status) {
        case 415:
            throw new APIError(ErrorCode.InvalidMimetype);
        case 424:
            throw new APIError(ErrorCode.DependencyMissing);
        case 403:
            if (accessToken != '') {
                throw new APIError(ErrorCode.PermissionDenied);
            }
    }

    if (!response.ok) {
        throw new APIError(ErrorCode.Unknown);
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

export const transformTimeStampedDates = <T>(item: T & TimeStamped) => {
    item.created_on = parseDateTime(item.created_on as unknown as string);
    item.updated_on = parseDateTime(item.created_on as unknown as string);
    if (item.valid_until != undefined) {
        item.valid_until = parseDateTime(item.valid_until as unknown as string);
    }
    return item;
};

export const transformTimeStampedDatesList = <T>(
    items: (T & TimeStamped)[]
) => {
    return items.map(transformTimeStampedDates);
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

export const renderDate = (d: Date | string): string => {
    switch (typeof d) {
        case 'object': {
            return format(d, 'yyyy-MM-dd');
        }
        case 'string':
            return format(parseISO(d), 'yyyy-MM-dd');
        default:
            return `${d}`;
    }
};

export interface QueryFilter<T> {
    offset?: number;
    limit?: number;
    desc?: boolean;
    query?: string;
    order_by?: keyof T;
    deleted?: boolean;
}

export interface AuthorQueryFilter<T> extends QueryFilter<T> {
    author_id?: string;
}

export interface ReportQueryFilter<T> extends AuthorQueryFilter<T> {
    report_status?: ReportStatus;
}
