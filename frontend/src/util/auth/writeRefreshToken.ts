import { refreshKey } from '../../auth.tsx';

export const writeRefreshToken = (token: string) => {
    if (token == '') {
        localStorage.removeItem(refreshKey);
    } else {
        localStorage.setItem(refreshKey, token);
    }
};
