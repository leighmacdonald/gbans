import { z } from 'zod';
import { PlayerProfile } from '../../api';
import { emptyOrNullString } from '../types.ts';
import { validateSteamID } from './validateSteamID.ts';

export const makeSteamidValidatorsOptional = (onSuccess?: (profile: PlayerProfile) => void) => {
    return {
        onChange: z.string().optional(),
        onChangeAsyncDebounceMs: 500,
        onChangeAsync: z.string().refine(
            async (value) => {
                if (emptyOrNullString(value)) {
                    return true;
                }
                const profile = await validateSteamID(value);
                if (!profile || !profile.player.steam_id) {
                    return false;
                }
                //TODO should this be done differently?
                onSuccess && onSuccess(profile);
                return true;
            },
            {
                message: 'SteamID / Profile invalid'
            }
        )
    };
};
