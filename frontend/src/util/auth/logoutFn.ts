import { apiCall, EmptyBody } from '../../api';
import { logoutKey, profileKey, refreshKey, tokenKey } from '../../auth.tsx';

export const logoutFn = async () => {
    localStorage.removeItem(profileKey);
    localStorage.removeItem(tokenKey);
    localStorage.removeItem(refreshKey);
    localStorage.setItem(logoutKey, Date.now().toString());
    sessionStorage.removeItem(tokenKey);
    await apiCall<EmptyBody>('/api/auth/logout', 'GET', undefined);
};
