import { ReportStatus } from './report';
import { format, parseISO } from 'date-fns';
import { applySteamId } from './profile';
import { readAccessToken, readRefreshToken, refreshToken } from './auth';

export enum PermissionLevel {
    Banned = 0,
    Guest = 1,
    User = 10,
    Editor = 25,
    Moderator = 50,
    Admin = 100
}

export interface apiError {
    error?: string;
}

export interface apiResponse<T> {
    status: boolean;
    message?: string;
    resp: Response;
    result?: T;
}
/**
 * All api requests are handled through this interface.
 *
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
    body?: TRequestBody,
    isRefresh?: boolean
): Promise<apiResponse<TResponse> & apiError> => {
    const headers: Record<string, string> = {
        'Content-Type': 'application/json; charset=UTF-8'
    };
    const opts: RequestInit = {
        mode: 'cors',
        credentials: 'include',
        method: method.toUpperCase()
    };
    const token = readAccessToken();
    const refresh = readRefreshToken();
    if (refresh != '' && token != '') {
        headers['Authorization'] = `Bearer ${token}`;
    }
    if (method !== 'GET' && body) {
        opts['body'] = JSON.stringify(body);
    }
    opts.headers = headers;
    const u = new URL(url, `${location.protocol}//${location.host}`);
    if (u.port == '8080') {
        u.port = '6006';
    }
    const resp = await fetch(u, opts);
    if (resp.status == 401 && !isRefresh && readRefreshToken() != '') {
        // Try and refresh the token once
        if (await refreshToken()) {
            // Successful token refresh, make a single recursive retry
            return apiCall(url, method, body, true);
        }
    }
    if (resp.status === 403 && token != '') {
        return { status: resp.ok, resp: resp, error: 'Unauthorized' };
    }
    const jsonText = await resp.text();
    const json: apiResponse<TResponse> = JSON.parse(jsonText, applySteamId);
    if (!resp.ok) {
        return {
            status: resp.ok && json.status,
            resp: resp,
            error: (json as apiError).error || ''
        };
    }
    return { result: json.result, resp, status: resp.ok && json.status };
};

export class ValidationException extends Error {}

export interface QueryFilterProps<T> {
    offset?: number;
    limit?: number;
    sort_desc?: boolean;
    query?: string;
    order_by?: keyof T;
    deleted?: boolean;
}

// Helper
export const StringIsNumber = (value: unknown) => !isNaN(Number(value));

export interface Pos {
    x: number;
    y: number;
    z: number;
}

export interface TimeStamped {
    created_on: Date;
    updated_on: Date;
    valid_until?: Date;
}

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
