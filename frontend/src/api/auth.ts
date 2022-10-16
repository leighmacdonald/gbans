import { apiCall } from './common';
import { logErr } from '../util/errors';

export const refreshKey = 'refresh';
export const tokenKey = 'token';
export const userKey = 'user';

export interface UserToken {
    token: string;
}

export const refreshToken = async () => {
    try {
        const resp = await apiCall<UserToken>('/api/auth/refresh', 'GET');
        if (!resp.status || !resp.result?.token) {
            logErr('Failed to refresh auth token');
            return resp;
        }
        localStorage.setItem(tokenKey, resp.result?.token);
        return resp;
    } catch (e) {
        logErr(e);
        return null;
    }
};
export const readToken = () => {
    try {
        return localStorage.getItem(tokenKey) as string;
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
