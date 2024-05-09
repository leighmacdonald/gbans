import { tokenKey } from '../../auth.tsx';

export const writeAccessToken = (token: string) => {
    if (token == '') {
        sessionStorage.removeItem(tokenKey);
    } else {
        sessionStorage.setItem(tokenKey, token);
    }
};
