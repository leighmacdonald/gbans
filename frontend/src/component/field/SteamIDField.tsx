import { QuestionMark } from '@mui/icons-material';
import CheckIcon from '@mui/icons-material/Check';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import HourglassBottomIcon from '@mui/icons-material/HourglassBottom';
import Avatar from '@mui/material/Avatar';
import InputAdornment from '@mui/material/InputAdornment';
import TextField, { TextFieldProps } from '@mui/material/TextField';
import { useStore } from '@tanstack/react-form';
import { defaultAvatarHash, PlayerProfile } from '../../api';
import { useFieldContext } from '../../contexts/formContext.tsx';
import { avatarHashToURL } from '../../util/text.tsx';

type Props = {
    profile?: PlayerProfile;
} & TextFieldProps;

export const SteamIDField = (props: Props) => {
    const field = useFieldContext<string>();
    const errors = useStore(field.store, (state) => state.meta.errors);

    return (
        <TextField
            {...props}
            value={field.state.value}
            onChange={(e) => field.handleChange(e.target.value)}
            onBlur={field.handleBlur}
            variant="filled"
            error={Boolean(errors)}
            helperText={errors ? errors.join(', ') : ''}
            slotProps={{
                input: {
                    startAdornment: (
                        <InputAdornment position="start">
                            {errors ? (
                                <ErrorOutlineIcon color={'error'} sx={{ width: 40 }} />
                            ) : field.state.meta.isValidating ? (
                                <HourglassBottomIcon color={'warning'} sx={{ width: 40 }} />
                            ) : field.state.meta.isTouched ? (
                                props.profile ? (
                                    <Avatar
                                        src={avatarHashToURL(props.profile?.player.avatarhash ?? defaultAvatarHash)}
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
                }
            }}
        />
    );
};
