import { accessTokenKey } from '../../auth.tsx';
import { logErr } from '../errors.ts';

export const readAccessToken = () => {
    try {
        return localStorage.getItem(accessTokenKey) as string;
    } catch (e) {
        logErr(e);
        return '';
    }
};
