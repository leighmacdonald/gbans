import { QuestionMark } from '@mui/icons-material';
import CheckIcon from '@mui/icons-material/Check';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import HourglassBottomIcon from '@mui/icons-material/HourglassBottom';
import Avatar from '@mui/material/Avatar';
import InputAdornment from '@mui/material/InputAdornment';
import TextField, { TextFieldProps } from '@mui/material/TextField';
import { defaultAvatarHash, PlayerProfile } from '../../api';
import { avatarHashToURL } from '../../util/text.tsx';
import { FieldProps } from './common.ts';

export const SteamIDField = ({
    defaultValue,
    handleBlur,
    handleChange,
    fullwidth,
    profile,
    error,
    helperText,
    isValidating,
    isTouched,
    label = 'SteamID/Profile'
}: FieldProps & { profile?: PlayerProfile } & TextFieldProps) => {
    return (
        <TextField
            fullWidth={fullwidth}
            label={label}
            defaultValue={defaultValue}
            onChange={(e) => handleChange(e.target.value)}
            onBlur={handleBlur}
            variant="filled"
            error={error}
            helperText={helperText}
            InputProps={{
                startAdornment: (
                    <InputAdornment position="start">
                        {error ? (
                            <ErrorOutlineIcon color={'error'} sx={{ width: 40 }} />
                        ) : isValidating ? (
                            <HourglassBottomIcon color={'warning'} sx={{ width: 40 }} />
                        ) : isTouched ? (
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
