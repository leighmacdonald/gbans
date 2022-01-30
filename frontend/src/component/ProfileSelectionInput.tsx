import TextField from '@mui/material/TextField';
import * as React from 'react';
import { apiGetProfile, PlayerProfile } from '../api';
import { log } from '../util/errors';
import { useState } from 'react';
import Stack from '@mui/material/Stack';
import Avatar from '@mui/material/Avatar';
import Typography from '@mui/material/Typography';
import InputAdornment from '@mui/material/InputAdornment';

export interface ProfileSelectionInputProps {
    renderFooter: boolean;
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
    fullWidth,
    renderFooter
}: ProfileSelectionInputProps) => {
    const [input, setInput] = useState<string>(initialValue ?? '');
    const [lProfile, setLProfile] = useState<PlayerProfile>();

    const loadProfile = async () => {
        try {
            const v = await apiGetProfile(input);
            onProfileSuccess(v);
            setLProfile(v);
        } catch (e) {
            log(e);
            setLProfile(undefined);
        }
    };

    const onChangeInput = (evt: React.ChangeEvent<HTMLInputElement>) => {
        const { value: nextValue } = evt.target;
        setInput(nextValue);
        loadProfile();
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
                            <Avatar
                                src={lProfile?.player.avatar}
                                variant={'square'}
                            />
                        </InputAdornment>
                    )
                }}
            />
            {renderFooter && (
                <Stack
                    spacing={3}
                    direction={'row'}
                    justifyContent="center"
                    alignItems="center"
                >
                    {lProfile?.player.steam_id ? (
                        <Avatar
                            variant={'square'}
                            alt="Avatar"
                            src={lProfile?.player.avatarfull}
                            sx={{ width: 56, height: 56 }}
                        />
                    ) : (
                        <Typography
                            variant={'subtitle1'}
                            alignContent={'center'}
                        >
                            Invalid Steam Profile
                        </Typography>
                    )}
                </Stack>
            )}
        </>
    );
};
