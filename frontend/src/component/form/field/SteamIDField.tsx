import { useCallback, useState } from 'react';
import { QuestionMark } from '@mui/icons-material';
import CheckIcon from '@mui/icons-material/Check';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import HourglassBottomIcon from '@mui/icons-material/HourglassBottom';
import Avatar from '@mui/material/Avatar';
import InputAdornment from '@mui/material/InputAdornment';
import TextField, { TextFieldProps } from '@mui/material/TextField';
import Tooltip from '@mui/material/Tooltip';
import { useStore } from '@tanstack/react-form';
import { debounce } from '@tanstack/react-pacer/debouncer';
import { useQueryClient } from '@tanstack/react-query';
import { apiGetSteamValidate, defaultAvatarHash, PlayerProfile, SteamValidate } from '../../../api';
import { useFieldContext } from '../../../contexts/formContext.tsx';
import { logErr } from '../../../util/errors.ts';
import { avatarHashToURL } from '../../../util/text.tsx';

type Props = {
    profile?: PlayerProfile;
} & TextFieldProps;

const empty = { steam_id: '', hash: '', personaname: '' };

export const SteamIDField = (props: Props) => {
    const [validation, setValidation] = useState<SteamValidate>(empty);
    const [error, setError] = useState<boolean>(false);
    const field = useFieldContext<string>();
    const [loading, setLoading] = useState<boolean>(false);
    const errors = useStore(field.store, (state) => state.meta.errors);
    const queryClient = useQueryClient();

    const onChange = useCallback(
        debounce(
            async (value) => {
                setLoading(true);
                try {
                    const data = await queryClient.fetchQuery({
                        queryKey: ['profile', value],
                        queryFn: async () => {
                            return await apiGetSteamValidate(value);
                        }
                    });
                    if (data.steam_id != '') {
                        setValidation(data);
                        field.setValue(data.steam_id);
                    }
                    setError(false);
                } catch (e) {
                    logErr(e);
                    setError(true);
                    setValidation(empty);
                } finally {
                    setLoading(false);
                }
            },
            {
                wait: 500
            }
        ),
        [setValidation]
    );

    return (
        <TextField
            {...props}
            fullWidth
            value={field.state.value}
            onChange={(e) => {
                setValidation(empty);
                onChange(e.target.value);
                field.handleChange(e.target.value);
            }}
            onBlur={field.handleBlur}
            variant="filled"
            error={errors.length > 0}
            helperText={
                errors.length > 0
                    ? errors.join(', ')
                    : (props.helperText ?? 'Can be any format of steam id or a profile URL.')
            }
            slotProps={{
                input: {
                    endAdornment: (
                        <InputAdornment position="end" variant={'filled'}>
                            {!field.state.meta.isTouched ? (
                                <QuestionMark color={'secondary'} />
                            ) : field.state.meta.isTouched &&
                              (error || (field.state.value != '' && errors.length > 0)) ? (
                                <ErrorOutlineIcon color={'error'} sx={{ width: 40 }} />
                            ) : loading || field.state.meta.isValidating ? (
                                <HourglassBottomIcon color={'warning'} sx={{ width: 40 }} />
                            ) : field.state.meta.isTouched && validation?.hash ? (
                                <Tooltip title={validation.personaname}>
                                    <Avatar
                                        src={avatarHashToURL(validation?.hash ?? defaultAvatarHash)}
                                        variant={'square'}
                                    />
                                </Tooltip>
                            ) : (
                                <CheckIcon color={'success'} />
                            )}
                        </InputAdornment>
                    )
                }
            }}
        />
    );
};
