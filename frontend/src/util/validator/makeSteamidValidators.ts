import { z } from 'zod';
import { PlayerProfile } from '../../api';
import { validateSteamID } from './validateSteamID.ts';

export const makeSteamidValidators = (onSuccess?: (profile: PlayerProfile) => void) => {
    return {
        onChange: z.string(),
        onChangeAsyncDebounceMs: 1000,
        onChangeAsync: z.string().refine(
            async (value) => {
                const profile = await validateSteamID(value);
                if (!profile || !profile.player.steam_id) {
                    return false;
                }
                //TODO should this be done differently?
                if (onSuccess) {
                    onSuccess(profile);
                }
                return true;
            },
            {
                message: 'SteamID / Profile invalid'
            }
        )
    };
};
