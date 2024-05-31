import { accessTokenKey } from '../../auth.tsx';

export const readAccessToken = () => {
    try {
        return localStorage.getItem(accessTokenKey) as string;
    } catch (e) {
        return '';
    }
};
