import TextField from '@mui/material/TextField';
import * as React from 'react';
import { apiGetProfile, PlayerProfile } from '../api';
import { log } from '../util/errors';
import { useState } from 'react';
import Stack from '@mui/material/Stack';
import Avatar from '@mui/material/Avatar';
import Typography from '@mui/material/Typography';
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
        }
    };

    const onChangeInput = (evt: React.ChangeEvent<HTMLInputElement>) => {
        const { value: nextValue } = evt.target;
        setInput(nextValue);
        loadProfile();
    };

    return (
        <>
            <TextField
                fullWidth={fullWidth}
                id={id ?? 'query'}
                label={label ?? 'Steam ID / Profile URL'}
                onChange={onChangeInput}
                onBlur={loadProfile}
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
