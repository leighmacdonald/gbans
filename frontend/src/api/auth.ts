import { apiCall, PermissionLevel } from './common';
import { logErr } from '../util/errors';

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
        localStorage.setItem('token', resp.result?.token);
        return resp;
    } catch (e) {
        logErr(e);
        return null;
    }
};

export const handleOnLogout = (): void => {
    localStorage.removeItem('token');
    localStorage.setItem('permission_level', `${PermissionLevel.Unknown}`);
    location.reload();
};

export const parseJwt = (token: string) => {
    const base64Payload = token.split('.')[1];
    const payload = Buffer.from(base64Payload, 'base64');
    return JSON.parse(payload.toString());
};
