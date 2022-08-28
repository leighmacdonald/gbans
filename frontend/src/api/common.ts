import { ReportStatus } from './report';
import { format, parseISO } from 'date-fns';
import { applySteamId } from './profile';
import { readToken } from './auth';

export enum PermissionLevel {
    Unknown = -1,
    Banned = 0,
    User = 1,
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
 */
export const apiCall = async <
    TResponse,
    TRequestBody = Record<string, unknown> | object
>(
    url: string,
    method: string,
    body?: TRequestBody
): Promise<apiResponse<TResponse> & apiError> => {
    const headers: Record<string, string> = {
        'Content-Type': 'application/json; charset=UTF-8'
    };
    const opts: RequestInit = {
        mode: 'cors',
        credentials: 'include',
        method: method.toUpperCase()
    };
    const token = readToken();
    if (token != '') {
        headers['Authorization'] = `Bearer ${token}`;
    }
    if (method !== 'GET' && body) {
        opts['body'] = JSON.stringify(body);
    }
    opts.headers = headers;
    const resp = await fetch(url, opts);
    if (resp.status === 403 && token != '') {
        return { status: resp.ok, resp: resp, error: 'Unauthorized' };
    }
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const jsonText = await resp.text();
    const json: apiResponse<TResponse> = JSON.parse(jsonText, applySteamId);
    if (!resp.ok) {
        return {
            status: resp.ok && json.status,
            resp: resp,
            error: (json as any).error || ''
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
