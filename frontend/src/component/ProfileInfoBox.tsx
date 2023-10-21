import React from 'react';
import PregnantWomanIcon from '@mui/icons-material/PregnantWoman';
import Avatar from '@mui/material/Avatar';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { format, fromUnixTime } from 'date-fns';
import { PlayerProfile } from '../api';
import { ContainerWithHeader, JustifyTypes } from './ContainerWithHeader';

export interface ProfileInfoBoxProps {
    profile: PlayerProfile;
    align?: JustifyTypes;
}

export const ProfileInfoBox = ({ profile, align }: ProfileInfoBoxProps) => {
    return (
        <ContainerWithHeader
            title={'Profile'}
            iconLeft={<PregnantWomanIcon />}
            align={align}
            marginTop={0}
        >
            <Stack direction={'row'} spacing={3} marginTop={0}>
                <Avatar
                    variant={'square'}
                    src={profile.player.avatarfull}
                    alt={'Profile Avatar'}
                    sx={{ width: 160, height: 160 }}
                />
                <Stack spacing={2} paddingTop={0}>
                    <Typography variant={'h1'}>
                        {profile.player.personaname}
                    </Typography>
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
