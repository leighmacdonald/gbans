import Stack from '@mui/material/Stack';
import Avatar from '@mui/material/Avatar';
import Typography from '@mui/material/Typography';
import { format, fromUnixTime } from 'date-fns';
import React from 'react';
import { PlayerProfile } from '../api';
import { ContainerWithHeader, JustifyTypes } from './ContainerWithHeader';
import PregnantWomanIcon from '@mui/icons-material/PregnantWoman';

export interface ProfileInfoBoxProps {
    profile: PlayerProfile;
    align?: JustifyTypes;
}

export const ProfileInfoBox = ({ profile, align }: ProfileInfoBoxProps) => {
    return (
        <ContainerWithHeader
            title={profile.player.personaname}
            iconLeft={<PregnantWomanIcon />}
            align={align}
        >
            <Stack direction={'row'} spacing={3}>
                <Avatar
                    variant={'square'}
                    src={profile.player.avatarfull}
                    alt={'Profile Avatar'}
                    sx={{ width: 184, height: 184 }}
                />
                <Stack spacing={2} paddingTop={3}>
                    <Typography variant={'subtitle1'}>
                        {profile.player.realname}
                    </Typography>
                    <Typography variant={'body1'}>
                        {[
                            profile.player.locstatecode,
                            profile.player.loccountrycode
                        ]
                            .filter((x) => x)
                            .join(',')}
                    </Typography>
                    <Typography variant={'body1'}>
                        Created:{' '}
                        {format(
                            fromUnixTime(profile.player.timecreated),
                            'yyyy-MM-dd'
                        )}
                    </Typography>
                </Stack>
            </Stack>
        </ContainerWithHeader>
    );
};
