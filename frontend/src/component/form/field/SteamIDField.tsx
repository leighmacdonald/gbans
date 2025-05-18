import { useMemo } from 'react';
import { QuestionMark } from '@mui/icons-material';
import CheckIcon from '@mui/icons-material/Check';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import HourglassBottomIcon from '@mui/icons-material/HourglassBottom';
import Avatar from '@mui/material/Avatar';
import InputAdornment from '@mui/material/InputAdornment';
import TextField, { TextFieldProps } from '@mui/material/TextField';
import { useStore } from '@tanstack/react-form';
import { defaultAvatarHash, PlayerProfile } from '../../../api';
import { useFieldContext } from '../../../contexts/formContext.tsx';
import { defaultFieldVariant } from '../../../theme.ts';
import { avatarHashToURL } from '../../../util/text.tsx';

type Props = {
    profile?: PlayerProfile;
} & TextFieldProps;

export const SteamIDField = (props: Props) => {
    const field = useFieldContext<string>();
    const errors = useStore(field.store, (state) => state.meta.errors);

    const adornment = useMemo(() => {
        if (field.state.meta.isValidating) {
            return <HourglassBottomIcon color={'warning'} sx={{ width: 40 }} />;
        }
        if (field.state.meta.isPristine) {
            return <QuestionMark color={'secondary'} />;
        }
        if (field.state.meta.errors.length > 0) {
            return <ErrorOutlineIcon color={'error'} sx={{ width: 40 }} />;
        }
        if (props.profile) {
            return (
                <Avatar
                    src={avatarHashToURL(props.profile?.player.avatarhash ?? defaultAvatarHash)}
                    variant={'square'}
                />
            );
        }

        return <CheckIcon color={'success'} />;
    }, [field.state.meta.isPristine, field.state.meta.isValidating, props.profile, field.state.meta.errors]);

    return (
        <TextField
            {...props}
            value={field.state.value}
            onChange={(e) => field.handleChange(e.target.value)}
            onBlur={field.handleBlur}
            variant={defaultFieldVariant}
            fullWidth
            error={errors.length > 0}
            helperText={errors ? errors.join(', ') : ''}
            slotProps={{
                input: {
                    startAdornment: <InputAdornment position={'end'}>{adornment}</InputAdornment>
                }
            }}
        />
    );
};
