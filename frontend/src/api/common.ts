import { ApiError } from '../error.tsx';
import { readAccessToken } from '../util/auth/readAccessToken.ts';
import { emptyOrNullString } from '../util/types';
import { AppealState } from './bans';

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

export type CallbackLink = {
    url: string;
};

export const apiRootURL = (): string => `${location.protocol}//${location.host}`;

type httpMethods = 'POST' | 'GET' | 'DELETE' | 'PUT';

/**
 * All api requests are handled through this interface.
 *
 * @param url
 * @param method
 * @param body
 * @param abortController
 * @param isFormData
 * @throws AppError
 */
export const apiCall = async <TResponse = EmptyBody | null, TRequestBody = Record<string, unknown> | object>(
    url: string,
    method: httpMethods = 'GET',
    body?: TRequestBody | undefined | FormData | Record<string, string>,
    abortController?: AbortController,
    isFormData: boolean = false
): Promise<TResponse> => {
    const headers: Record<string, string> = {};
    const requestOptions: RequestInit = {
        mode: 'cors',
        credentials: 'include',
        method: method.toUpperCase()
    };

    const accessToken = readAccessToken();

    if (!emptyOrNullString(accessToken)) {
        headers['Authorization'] = `Bearer ${accessToken}`;
    }

    if (!isFormData) {
        headers['Content-Type'] = 'application/json; charset=UTF-8';
    }

    requestOptions.headers = headers;

    if (method !== 'GET' && body) {
        requestOptions.body = isFormData ? (body as FormData) : JSON.stringify(body);
    }

    if (abortController != undefined) {
        requestOptions.signal = abortController.signal;
    }

    const fullURL = new URL(url, apiRootURL());
    if (method == 'GET' && body) {
        fullURL.search = new URLSearchParams(body as Record<string, string>).toString();
    }

    const response = await fetch(fullURL, requestOptions);

    if (!response.ok) {
        throw (await response.json()) as ApiError;
    }

    if (response.status == 204) {
        return null as TResponse;
    }

    return (await response.json()) as TResponse;
};

export interface QueryFilter {
    offset?: number;
    limit?: number;
    desc?: boolean;
    query?: string;
    order_by?: string;
    deleted?: boolean;
    flagged_only?: boolean;
}

export interface BanQueryCommon extends QueryFilter {
    source_id?: string;
    target_id?: string;
    appeal_state?: AppealState;
    deleted?: boolean;
}

export type BanSteamQueryFilter = BanQueryCommon;

export interface BanCIDRQueryFilter extends BanQueryCommon {
    ip?: string;
}

export interface BanGroupQueryFilter extends BanQueryCommon {
    group_id?: string;
}

export interface BanASNQueryFilter extends BanQueryCommon {
    as_num?: number;
}

export interface ReportQueryFilter {
    deleted?: boolean;
    source_id?: string;
}
