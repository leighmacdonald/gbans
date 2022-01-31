import TextField from '@mui/material/TextField';
import * as React from 'react';
import { apiGetProfile, PlayerProfile } from '../api';
import { log } from '../util/errors';
import { ChangeEvent, useState } from 'react';
import Avatar from '@mui/material/Avatar';
import InputAdornment from '@mui/material/InputAdornment';
import HourglassBottomIcon from '@mui/icons-material/HourglassBottom';
import { useTimer } from 'react-timer-hook';
import ErrorOutlineIcon from '@mui/icons-material/ErrorOutline';

export interface ProfileSelectionInputProps {
    id?: string;
    label?: string;
    initialValue?: string;
    fullWidth: boolean;
    onProfileSuccess: (profile: PlayerProfile) => void;
}

export const ProfileSelectionInput = ({
    onProfileSuccess,
    id,
    initialValue,
    label,
    fullWidth
}: ProfileSelectionInputProps) => {
    const rate = 1;
    const [loading, setLoading] = useState<boolean>(false);
    const [input, setInput] = useState<string>(initialValue ?? '');
    const [lProfile, setLProfile] = useState<PlayerProfile>();
    const { restart, pause } = useTimer({
        expiryTimestamp: new Date(),
        autoStart: true,
        onExpire: async () => {
            await loadProfile();
        }
    });
    const loadProfile = async () => {
        try {
            const v = await apiGetProfile(input);
            onProfileSuccess(v);
            setLProfile(v);
        } catch (e) {
            log(e);
            setLProfile(undefined);
        }
        setLoading(false);
    };

    const onChangeInput = (evt: ChangeEvent<HTMLInputElement>) => {
        const { value: nextValue } = evt.target;
        setInput(nextValue);
        if (nextValue == '') {
            setLoading(false);
            pause();
            return;
        }
        setLoading(true);
        const time = new Date();
        time.setSeconds(time.getSeconds() + rate);
        restart(time);
    };

    const isError =
        input.length > 0 && (!lProfile || !lProfile?.player.steam_id);
    return (
        <>
            <TextField
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
