import { apiGetProfile } from '../../api';
import { logErr } from '../errors.ts';

export const validateSteamID = async (arg: string | undefined) => {
    if (!arg) {
        return '';
    }
    try {
        return await apiGetProfile(arg);
    } catch (e) {
        logErr(e);
        return '';
    }
};
