import Stack from '@mui/material/Stack';
import Avatar from '@mui/material/Avatar';
import Typography from '@mui/material/Typography';
import { format, fromUnixTime } from 'date-fns';
import Paper from '@mui/material/Paper';
import React from 'react';
import { PlayerProfile } from '../api';
import { Heading } from './Heading';

export interface ProfileInfoBoxProps {
    profile: PlayerProfile;
}

export const ProfileInfoBox = ({ profile }: ProfileInfoBoxProps) => {
    return (
        <Paper elevation={1} sx={{ width: '100%' }}>
            <Heading>{profile.player.personaname}</Heading>
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
                        {profile.player.locstatecode},{' '}
                        {profile.player.loccountrycode}
                    </Typography>
                    <Typography variant={'body1'}>
                        Created:{' '}
                        {format(
                            fromUnixTime(profile.player.timecreated),
                            'yyyy-mm-dd'
                        )}
                    </Typography>
                </Stack>
            </Stack>
        </Paper>
    );
};
