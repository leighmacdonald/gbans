import { accessTokenKey } from '../../auth.tsx';

export const writeAccessToken = (token: string) => {
    if (token == '') {
        localStorage.removeItem(accessTokenKey);
    } else {
        localStorage.setItem(accessTokenKey, token);
    }
};
