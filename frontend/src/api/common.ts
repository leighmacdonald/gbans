import { ReportStatus } from './report';
import { format, parseISO } from 'date-fns';
import {
    isTokenExpired,
    readAccessToken,
    readRefreshToken,
    refreshToken
} from './auth';
import { parseDateTime } from '../util/text';
import { MatchResult } from './match';

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

/**
 * All api requests are handled through this interface.
 *
 * @param objectType
 * @param url
 * @param method
 * @param body
 * @param isRefresh
 */
export const apiCall = async <
    TResponse,
    TRequestBody = Record<string, unknown> | object
>(
    url: string,
    method: string,
    body?: TRequestBody | undefined,
    isRefresh?: boolean
): Promise<TResponse> => {
    const headers: Record<string, string> = {
        'Content-Type': 'application/json; charset=UTF-8'
    };
    const opts: RequestInit = {
        mode: 'cors',
        credentials: 'include',
        method: method.toUpperCase()
    };

    let token = readAccessToken();
    const refresh = readRefreshToken();
    if (token == '' || isTokenExpired(token)) {
        if (refresh != '' && !isTokenExpired(refresh)) {
            token = await refreshToken();
        }
    }

    if (isRefresh) {
        // Use the refresh token instead when performing a token refresh request
        token = readRefreshToken();
    }

    if (token != '') {
        headers['Authorization'] = `Bearer ${token}`;
    }

    if (method !== 'GET' && body) {
        opts['body'] = JSON.stringify(body);
    }
    opts.headers = headers;
    const u = new URL(url, `${location.protocol}//${location.host}`);
    const resp = await fetch(u, opts);

    if (resp.status == 401 && !isRefresh && refresh != '' && token != '') {
        // Try and refresh the token once
        if ((await refreshToken()) != '') {
            // Successful token refresh, make a single recursive retry
            return apiCall(url, method, body, false);
        }
    }
    if (resp.status === 403 && token != '') {
        throw new Error('Permission Denied');
    }

    if (!resp.ok) {
        throw new Error(`Invalid response`);
    }

    return (await resp.json()) as TResponse;
};

export class ValidationException extends Error {}

export interface MatchTimes {
    time_start: Date;
    time_end: Date;
}

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
