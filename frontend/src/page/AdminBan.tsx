import React, { useState } from 'react';
import Grid from '@mui/material/Grid';
import Paper from '@mui/material/Paper';
import Typography from '@mui/material/Typography';
import { PlayerBanForm } from '../component/PlayerBanForm';
import { ProfilePanel } from '../component/ProfilePanel';
import { PlayerProfile } from '../api';
import Box from '@mui/material/Box';
import { Nullable } from '../util/types';

export const AdminBan = (): JSX.Element => {
    const [profile, setProfile] = useState<Nullable<PlayerProfile>>();
    return (
        <Grid container spacing={3} paddingTop={3}>
            <Grid item xs={6}>
                <Paper elevation={1}>
                    <Box padding={3}>
                        <Typography variant={'h1'}>Ban A Player</Typography>
                    </Box>
                    <PlayerBanForm
                        onProfileChanged={(p) => {
                            setProfile(p);
                        }}
                    />
                </Paper>
            </Grid>
            <Grid item xs={6}>
                <Paper elevation={1}>
                    <Box padding={3}>
                        <Typography variant={'h1'}>Player Profile</Typography>
                    </Box>
                    <ProfilePanel profile={profile} />
                </Paper>
            </Grid>
        </Grid>
    );
};
