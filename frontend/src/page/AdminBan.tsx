import Grid from '@mui/material/Grid';
import Paper from '@mui/material/Paper';
import React, { useState } from 'react';
import { PlayerBanForm } from '../component/PlayerBanForm';
import { ProfilePanel } from '../component/ProfilePanel';
import { PlayerProfile } from '../util/api';
import { Typography } from '@mui/material';

export const AdminBan = (): JSX.Element => {
    const [profile, setProfile] = useState<PlayerProfile | undefined>();
    return (
        <Grid container spacing={3}>
            <Grid item xs={6}>
                <Paper>
                    <Grid item xs={12}>
                        <Typography variant={'h1'}>Ban A Player</Typography>
                    </Grid>
                    <PlayerBanForm
                        onProfileChanged={(p) => {
                            setProfile(p);
                        }}
                    />
                </Paper>
            </Grid>
            <Grid item xs={6}>
                <Paper>
                    <Grid item xs={12}>
                        <Typography variant={'h1'}>Player Profile</Typography>
                    </Grid>
                    <ProfilePanel profile={profile} />
                </Paper>
            </Grid>
        </Grid>
    );
};
