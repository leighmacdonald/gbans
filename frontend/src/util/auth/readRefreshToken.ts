import { refreshKey } from '../../auth.tsx';

export const readRefreshToken = () => {
    try {
        return localStorage.getItem(refreshKey) as string;
    } catch (e) {
        return '';
    }
};
