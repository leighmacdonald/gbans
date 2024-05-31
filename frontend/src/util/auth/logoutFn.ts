import { logoutKey, profileKey, accessTokenKey } from '../../auth.tsx';

export const logoutFn = async () => {
    // try {
    //     await apiCall<EmptyBody>('/api/auth/logout', 'GET', undefined);
    // } catch (e) {
    //     console.log(`Error calling logout handler:${e}`);
    // }
    localStorage.removeItem(profileKey);
    localStorage.removeItem(accessTokenKey);
    localStorage.setItem(logoutKey, Date.now().toString());
};
