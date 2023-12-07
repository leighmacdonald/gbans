import React, { ChangeEvent, useCallback, useState } from 'react';
import { useTimer } from 'react-timer-hook';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import HourglassBottomIcon from '@mui/icons-material/HourglassBottom';
import Avatar from '@mui/material/Avatar';
import InputAdornment from '@mui/material/InputAdornment';
import TextField from '@mui/material/TextField';
import { useFormikContext } from 'formik';
import { apiGetProfile, PlayerProfile } from '../api';
import { logErr } from '../util/errors';
import { avatarHashToURL } from '../util/text';
import { Nullable } from '../util/types';

export interface ProfileSelectionInputProps {
    label?: string;
    initialValue?: string;
}

interface ProfileSelectionFieldProps {
    steam_id: string;
}

export const ProfileSelectionField = <T,>({
    label
}: ProfileSelectionInputProps) => {
    const debounceRate = 1;
    const [input, setInput] = useState('');
    const [loading, setLoading] = useState<boolean>(false);
    const [lProfile, setLProfile] = useState<Nullable<PlayerProfile>>();

    const { setFieldValue, touched, errors } = useFormikContext<
        T & ProfileSelectionFieldProps
    >();

    const loadProfile = useCallback(async () => {
        if (input) {
            setLoading(true);
            try {
                const resp = await apiGetProfile(input);
                await setFieldValue('steam_id', resp?.player.steam_id);
                setLProfile(resp);
                setLoading(false);
            } catch (e) {
                setLProfile(undefined);
                logErr(e);
            } finally {
                setLoading(false);
            }
        }
    }, [input, setFieldValue]);

    const { restart, pause } = useTimer({
        expiryTimestamp: new Date(),
        autoStart: true,
        onExpire: loadProfile
    });

    const onChangeInput = (evt: ChangeEvent<HTMLInputElement>) => {
        const { value: nextValue } = evt.target;
        setInput(nextValue);
        if (nextValue == '') {
            setLoading(false);
            setLProfile(null);
            pause();
            return;
        }
        setLoading(true);
        const time = new Date();
        time.setSeconds(time.getSeconds() + debounceRate);
        restart(time);
    };

    return (
        <>
            <TextField
                value={input}
                fullWidth
                id={'steam_id'}
                name={'steam_id'}
                label={label ?? 'Steam ID / Profile URL'}
                onChange={onChangeInput}
                onBlur={loadProfile}
                color={lProfile?.player.steam_id ? 'success' : 'primary'}
                error={touched.steam_id && Boolean(errors.steam_id)}
                InputProps={{
                    startAdornment: (
                        <InputAdornment position="start">
                            {touched.steam_id && Boolean(errors.steam_id) ? (
                                <ErrorOutlineIcon
                                    color={'error'}
                                    sx={{ width: 40 }}
                                />
                            ) : loading ? (
                                <HourglassBottomIcon sx={{ width: 40 }} />
                            ) : (
                                <Avatar
                                    src={avatarHashToURL(
                                        lProfile?.player.avatarhash
                                    )}
                                    variant={'square'}
                                />
                            )}
                        </InputAdornment>
                    )
                }}
            />
        </>
    );
};
