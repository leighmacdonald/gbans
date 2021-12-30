import React from 'react';
import { StatsPanel } from '../component/StatsPanel';
import { BanList } from '../component/BanList';
import { ServerList } from '../component/ServerList';
import Grid from '@mui/material/Grid';
import { Paper, Typography } from '@mui/material';

export const Home = (): JSX.Element => {
    return (
        <Grid container spacing={3}>
            <Grid item xs={9}>
                <Paper>
                    <Typography variant={'h2'}>Most Recent Bans</Typography>
                    <BanList />
                </Paper>
            </Grid>
            <Grid item xs={3}>
                <Paper>
                    <Typography variant={'h2'}>DB Stats</Typography>
                    <StatsPanel />
                </Paper>
            </Grid>
            <Grid item xs={12}>
                <Paper>
                    <Typography variant={'h2'}>Server List</Typography>
                    <ServerList />
                </Paper>
            </Grid>
        </Grid>
    );
};
