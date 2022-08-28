import React, { ChangeEvent, useState } from 'react';
import Avatar from '@mui/material/Avatar';
import InputAdornment from '@mui/material/InputAdornment';
import HourglassBottomIcon from '@mui/icons-material/HourglassBottom';
import TextField from '@mui/material/TextField';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';
import { useTimer } from 'react-timer-hook';
import { apiGetProfile, PlayerProfile } from '../api';
import { logErr } from '../util/errors';
import { Nullable } from '../util/types';

export interface ProfileSelectionInputProps {
    id?: string;
    label?: string;
    initialValue?: string;
    fullWidth: boolean;
    onProfileSuccess: (profile: Nullable<PlayerProfile>) => void;
    input: string;
    setInput: (input: string) => void;
}

export const ProfileSelectionInput = ({
    onProfileSuccess,
    id,
    label,
    fullWidth,
    input,
    setInput
}: ProfileSelectionInputProps) => {
    const debounceRate = 1;
    const [loading, setLoading] = useState<boolean>(false);
    const [lProfile, setLProfile] = useState<Nullable<PlayerProfile>>();

    const loadProfile = () => {
        if (input) {
            setLoading(true);
            apiGetProfile(input)
                .then((response) => {
                    if (!response.status || !response.result) {
                        setLProfile(undefined);
                        return;
                    }
                    onProfileSuccess(response.result);
                    setLProfile(response.result);
                    setLoading(false);
                })
                .catch((e) => {
                    setLProfile(undefined);
                    logErr(e);
                });
        }
    };

    const { restart, pause } = useTimer({
        expiryTimestamp: new Date(),
        autoStart: true,
        onExpire: loadProfile
    });

    const onChangeInput = (evt: ChangeEvent<HTMLInputElement>) => {
        const { value: nextValue } = evt.target;
        setInput(nextValue);
        if (nextValue == '') {
            onProfileSuccess(null);
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

    const isError =
        input != '' && !loading && (!lProfile || !lProfile?.player.steam_id);
    return (
        <>
            <TextField
                value={input}
                error={isError}
                fullWidth={fullWidth}
                id={id ?? 'query'}
                label={label ?? 'Steam ID / Profile URL'}
                onChange={onChangeInput}
                onBlur={loadProfile}
                color={lProfile?.player.steam_id ? 'success' : 'primary'}
                InputProps={{
                    startAdornment: (
                        <InputAdornment position="start">
                            {isError ? (
                                <ErrorOutlineIcon
                                    color={'error'}
                                    sx={{ width: 40 }}
                                />
                            ) : loading ? (
                                <HourglassBottomIcon sx={{ width: 40 }} />
                            ) : (
                                <Avatar
                                    src={lProfile?.player.avatar}
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
