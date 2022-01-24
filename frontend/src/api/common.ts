import { log } from '../util/errors';

export enum PermissionLevel {
    Guest = 1,
    Banned = 2,
    Authenticated = 10,
    Moderator = 50,
    Admin = 100
}

export interface apiResponse<T> {
    status: boolean;
    resp: Response;
    json: T | apiError;
}

export interface apiError {
    error?: string;
}

export const apiCall = async <
    TResponse,
    TRequestBody = Record<string, unknown>
>(
    url: string,
    method: string,
    body?: TRequestBody
): Promise<apiResponse<TResponse>> => {
    const headers: Record<string, string> = {
        'Content-Type': 'application/json; charset=UTF-8'
    };
    const opts: RequestInit = {
        mode: 'cors',
        credentials: 'include',
        method: method.toUpperCase()
    };
    const token = localStorage.getItem('token');
    if (token != '') {
        headers['Authorization'] = `Bearer ${token}`;
    }
    if (method === 'POST' && body) {
        opts['body'] = JSON.stringify(body);
    }
    opts.headers = headers;
    const resp = await fetch(url, opts);
    if (resp.status === 403 && token != '') {
        log('invalid token');
    }
    if (!resp.status) {
        throw apiErr('Invalid response code', resp);
    }
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const json = ((await resp.json()) as TResponse as any).data;
    if (json?.error && json.error !== '') {
        throw apiErr(`Error received: ${json.error}`, resp);
    }
    return { json: json, resp: resp, status: resp.ok };
};

class ApiException extends Error {
    public resp: Response;

    constructor(msg: string, response: Response) {
        super(msg);
        this.resp = response;
    }
}

const apiErr = (msg: string, resp: Response): ApiException => {
    return new ApiException(msg, resp);
};

export interface QueryFilterProps {
    offset: number;
    limit: number;
    sort_desc: boolean;
    query: string;
    order_by: string;
}

export const handleOnLogin = (): void => {
    let returnUrl = window.location.hostname;
    if (
        (window.location.protocol === 'https:' &&
            window.location.port !== '443') ||
        (window.location.protocol === 'http:' &&
            window.location.port !== '80') ||
        (window.location.port != '80' && window.location.port != '443')
    ) {
        returnUrl = `${returnUrl}:${window.location.port}`;
    }
    const r = `${window.location.protocol}//${returnUrl}/auth/callback?return_url=${window.location.pathname}`;
    const oid =
        'https://steamcommunity.com/openid/login' +
        '?openid.ns=' +
        encodeURIComponent('http://specs.openid.net/auth/2.0') +
        '&openid.mode=checkid_setup' +
        '&openid.return_to=' +
        encodeURIComponent(r) +
        `&openid.realm=` +
        encodeURIComponent(
            `${window.location.protocol}//${window.location.hostname}`
        ) +
        '&openid.ns.sreg=' +
        encodeURIComponent('http://openid.net/extensions/sreg/1.1') +
        '&openid.claimed_id=' +
        encodeURIComponent(
            'http://specs.openid.net/auth/2.0/identifier_select'
        ) +
        '&openid.identity=' +
        encodeURIComponent(
            'http://specs.openid.net/auth/2.0/identifier_select'
        );
    window.open(oid, '_self');
};

export const handleOnLogout = (): void => {
    localStorage.removeItem('token');
    location.reload();
};

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
}
