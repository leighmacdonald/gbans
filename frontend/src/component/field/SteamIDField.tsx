import { QuestionMark } from '@mui/icons-material';
import CheckIcon from '@mui/icons-material/Check';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import HourglassBottomIcon from '@mui/icons-material/HourglassBottom';
import Avatar from '@mui/material/Avatar';
import InputAdornment from '@mui/material/InputAdornment';
import TextField from '@mui/material/TextField';
import { defaultAvatarHash, PlayerProfile } from '../../api';
import { avatarHashToURL } from '../../util/text.tsx';
import { FieldProps } from './common.ts';

export const SteamIDField = ({
    state,
    handleBlur,
    handleChange,
    fullwidth,
    profile,
    label = 'SteamID/Profile'
}: FieldProps & { profile?: PlayerProfile }) => {
    return (
        <TextField
            fullWidth={fullwidth}
            label={label}
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
                            <HourglassBottomIcon color={'warning'} sx={{ width: 40 }} />
                        ) : state.meta.isTouched ? (
                            profile ? (
                                <Avatar
                                    src={avatarHashToURL(profile?.player.avatarhash ?? defaultAvatarHash)}
                                    variant={'square'}
                                />
                            ) : (
                                <CheckIcon color={'success'} />
                            )
                        ) : (
                            <QuestionMark color={'secondary'} />
                        )}
                    </InputAdornment>
                )
            }}
        />
    );
};
