import { tokenKey } from '../../auth.tsx';

export const readAccessToken = () => {
    try {
        return sessionStorage.getItem(tokenKey) as string;
    } catch (e) {
        return '';
    }
};
