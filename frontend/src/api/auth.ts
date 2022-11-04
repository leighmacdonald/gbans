import { apiCall } from './common';
import { logErr } from '../util/errors';

export const refreshKey = 'refresh';
export const tokenKey = 'token';
export const userKey = 'user';

export interface UserToken {
    accessToken: string;
    refreshToken: string;
}

export const refreshToken = async () => {
    try {
        const resp = await apiCall<UserToken>(
            '/api/auth/refresh',
            'POST',
            {
                refreshToken: readRefreshToken()
            } as UserToken,
            true
        );
        if (
            !resp.status ||
            !resp.result?.accessToken ||
            !resp.result?.refreshToken
        ) {
            logErr('Failed to refresh auth token');
            return '';
        }
        writeAccessToken(resp.result?.accessToken);
        writeRefreshToken(resp.result?.refreshToken);
        return resp.result?.accessToken;
    } catch (e) {
        logErr(e);
        return '';
    }
};

export const writeAccessToken = (token: string) => {
    try {
        return sessionStorage.setItem(tokenKey, token);
    } catch (e) {
        return '';
    }
};

export const readAccessToken = () => {
    try {
        return sessionStorage.getItem(tokenKey) as string;
    } catch (e) {
        return '';
    }
};

export const writeRefreshToken = (token: string) => {
    try {
        return localStorage.setItem(refreshKey, token);
    } catch (e) {
        return '';
    }
};

export const readRefreshToken = () => {
    try {
        return localStorage.getItem(refreshKey) as string;
    } catch (e) {
        return '';
    }
};

// export const handleOnLogout = (): void => {
//     localStorage.clear();
//     location.reload();
// };

export const parseJwt = (token: string) => {
    const base64Payload = token.split('.')[1];
    const payload = Buffer.from(base64Payload, 'base64');
    return JSON.parse(payload.toString());
};

export const handleOnLogin = (returnPath: string): string => {
    let returnUrl = window.location.hostname;
    if (window.location.port !== '') {
        returnUrl = `${returnUrl}:${window.location.port}`;
    }
    // Don't redirect loop to /login
    const r = `${
        window.location.protocol
    }//${returnUrl}/auth/callback?return_url=${
        returnPath !== '/login' ? returnPath : '/'
    }`;
    return (
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
        encodeURIComponent('http://specs.openid.net/auth/2.0/identifier_select')
    );
};
