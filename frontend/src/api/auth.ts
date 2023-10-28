import decodeJWT, { JwtPayload } from 'jwt-decode';
import { logErr } from '../util/errors';
import { apiCall, EmptyBody } from './common';

export const refreshKey = 'refresh';
export const tokenKey = 'token';
export const userKey = 'user';
export const logoutKey = 'logout';

export interface UserToken {
    access_token: string;
    refresh_token: string;
}

export const refreshToken = async () => {
    try {
        const resp = await apiCall<UserToken>(
            '/api/auth/refresh',
            'POST',
            {
                refresh_token: readRefreshToken()
            } as UserToken,
            undefined,
            true
        );
        if (!resp?.access_token) {
            logErr('Failed to refresh auth token');
            return;
        }
        writeAccessToken(resp?.access_token);
    } catch (e) {
        logErr(e);
    }
};

export const isTokenExpired = (token: string): boolean => {
    if (!token || token == '') {
        return true;
    }

    const claims: JwtPayload = decodeJWT(token);
    if (!claims || !claims.exp) {
        return true;
    }

    const expirationTimeInSeconds = claims.exp * 1000;
    const now = new Date();

    return expirationTimeInSeconds <= now.getTime();
};

export const writeAccessToken = (token: string) => {
    if (token == '') {
        sessionStorage.removeItem(tokenKey);
    } else {
        sessionStorage.setItem(tokenKey, token);
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
    if (token == '') {
        localStorage.removeItem(refreshKey);
    } else {
        localStorage.setItem(refreshKey, token);
    }
};

export const readRefreshToken = () => {
    try {
        return localStorage.getItem(refreshKey) as string;
    } catch (e) {
        return '';
    }
};

// Calling writeLogoutKey will trigger a logout on any other currently open tabs using
// a storage event listener. See: LogoutHandler.tsx
export const writeLogoutKey = () => {
    localStorage.setItem(logoutKey, Date.now().toString());
};

export const logout = async (abortController?: AbortController) => {
    try {
        await apiCall<EmptyBody>(
            '/api/auth/logout',
            'GET',
            undefined,
            abortController
        );
    } catch (e) {
        logErr(e);
    } finally {
        writeAccessToken('');
        writeRefreshToken('');
        writeLogoutKey();
    }
};

export const parseJwt = (token: string) => {
    const base64Payload = token.split('.')[1];
    const payload = Buffer.from(base64Payload, 'base64');
    return JSON.parse(payload.toString());
};

const baseUrl = () => {
    let returnUrl = window.location.hostname;
    if (window.location.port !== '') {
        returnUrl = `${returnUrl}:${window.location.port}`;
    }
    return `${window.location.protocol}//${returnUrl}`;
};

export const handleOnLogin = (returnPath: string): string => {
    let returnUrl = window.location.hostname;
    if (window.location.port !== '') {
        returnUrl = `${returnUrl}:${window.location.port}`;
    }
    // Don't redirect loop to /login
    const returnTo = `${
        window.location.protocol
    }//${returnUrl}/auth/callback?return_url=${
        returnPath !== '/login' ? returnPath : '/'
    }`;

    return [
        'https://steamcommunity.com/openid/login',
        `?openid.ns=${encodeURIComponent('http://specs.openid.net/auth/2.0')}`,
        '&openid.mode=checkid_setup',
        `&openid.return_to=${encodeURIComponent(returnTo)}`,
        `&openid.realm=${encodeURIComponent(
            `${window.location.protocol}//${window.location.hostname}`
        )}`,
        `&openid.ns.sreg=${encodeURIComponent(
            'http://openid.net/extensions/sreg/1.1'
        )}`,
        `&openid.claimed_id=${encodeURIComponent(
            'http://specs.openid.net/auth/2.0/identifier_select'
        )}`,
        `&openid.identity=${encodeURIComponent(
            'http://specs.openid.net/auth/2.0/identifier_select'
        )}`
    ].join('');
};

export const discordLoginURL = () => {
    return (
        'https://discord.com/oauth2/authorize' +
        '?client_id=' +
        window.gbans.discord_client_id +
        '&redirect_uri=' +
        encodeURIComponent(baseUrl() + '/login/discord') +
        '&response_type=code' +
        '&scope=identify'
    );
};
