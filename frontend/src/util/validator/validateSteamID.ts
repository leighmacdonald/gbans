import { apiGetProfile } from '../../api';

export const validateSteamID = async (arg: string | undefined) => {
    if (!arg) {
        return '';
    }
    try {
        return await apiGetProfile(arg);
    } catch (e) {
        return '';
    }
};
