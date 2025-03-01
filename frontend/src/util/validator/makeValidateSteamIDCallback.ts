import { z } from 'zod';
import { PlayerProfile } from '../../api';
import { emptyOrNullString } from '../types.ts';
import { validateSteamID } from './validateSteamID.ts';

export const makeValidateSteamIDCallback = (onSuccess?: (profile: PlayerProfile) => void) => {
    return z.string().refine(
        async (value?: string): Promise<boolean> => {
            if (emptyOrNullString(value)) {
                return true;
            }

            const profile = await validateSteamID(value);
            if (onSuccess && profile) {
                onSuccess(profile);
            }

            return !(!profile || !profile.player.steam_id);
        },
        {
            message: 'SteamID / Profile link invalid'
        }
    );
};
