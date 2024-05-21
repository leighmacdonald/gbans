import { apiCall, EmptyBody } from '../../api';
import { logoutKey, profileKey, refreshKey, tokenKey } from '../../auth.tsx';

export const logoutFn = async () => {
    try {
        await apiCall<EmptyBody>('/api/auth/logout', 'GET', undefined);
    } catch (e) {
        console.log(`Error calling logout handler:${e}`);
    }
    localStorage.removeItem(profileKey);
    localStorage.removeItem(tokenKey);
    localStorage.removeItem(refreshKey);
    localStorage.setItem(logoutKey, Date.now().toString());
    sessionStorage.removeItem(tokenKey);
};
