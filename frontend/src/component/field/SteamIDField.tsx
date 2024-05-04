import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import HourglassBottomIcon from '@mui/icons-material/HourglassBottom';
import Avatar from '@mui/material/Avatar';
import InputAdornment from '@mui/material/InputAdornment';
import TextField from '@mui/material/TextField';
import { z } from 'zod';
import { apiGetProfile, defaultAvatarHash, PlayerProfile } from '../../api';
import { avatarHashToURL } from '../../util/text.tsx';
import { emptyOrNullString } from '../../util/types.ts';
import { FieldProps } from './common.ts';

const validateSteamID = async (arg: string | undefined) => {
    if (!arg) {
        return '';
    }
    try {
        return await apiGetProfile(arg);
    } catch (e) {
        return '';
    }
};

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
                onSuccess && onSuccess(profile);
                return true;
            },
            {
                message: 'SteamID / Profile invalid'
            }
        )
    };
};

export const makeSteamidValidatorsOptional = (onSuccess?: (profile: PlayerProfile) => void) => {
    return {
        onChange: z.string().optional(),
        onChangeAsyncDebounceMs: 1000,
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

export const SteamIDField = ({ state, handleBlur, handleChange, fullwidth, profile }: FieldProps<string> & { profile?: PlayerProfile }) => {
    return (
        <TextField
            fullWidth={fullwidth}
            label="SteamID/Profile"
            defaultValue={state.value}
            onChange={(e) => handleChange(e.target.value)}
            onBlur={handleBlur}
            variant="outlined"
            error={state.meta.touchedErrors.length > 0}
            helperText={state.meta.touchedErrors}
            InputProps={{
                startAdornment: (
                    <InputAdornment position="start">
                        {state.meta.touchedErrors.length > 0 ? (
                            <ErrorOutlineIcon color={'error'} sx={{ width: 40 }} />
                        ) : state.meta.isValidating ? (
                            <HourglassBottomIcon sx={{ width: 40 }} />
                        ) : (
                            <Avatar src={avatarHashToURL(profile?.player.avatarhash ?? defaultAvatarHash)} variant={'square'} />
                        )}
                    </InputAdornment>
                )
            }}
        />
    );
};
