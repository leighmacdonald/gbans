import 'core-js/stable/atob';
import { jwtDecode, JwtPayload } from 'jwt-decode';
import { logoutFn, readRefreshToken, writeAccessToken } from '../auth.tsx';
import { logErr } from '../util/errors';
import { emptyOrNullString } from '../util/types';
import { apiCall } from './common';

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
        writeAccessToken(resp.access_token);
    } catch (e) {
        logErr(e);
        await logoutFn();
    }
};

export const isTokenExpired = (token: string): boolean => {
    if (emptyOrNullString(token)) {
        return true;
    }

    const now = new Date();

    try {
        const claims: JwtPayload = jwtDecode(token);
        if (!claims || !claims.exp) {
            return true;
        }

        const expirationTimeInSeconds = claims.exp * 1000;
        return expirationTimeInSeconds <= now.getTime();
    } catch (e) {
        return true;
    }
};
