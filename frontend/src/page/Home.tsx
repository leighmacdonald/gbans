import React from 'react';
import { StatsPanel } from '../component/StatsPanel';
import { BanList } from '../component/BanList';
import { ServerList } from '../component/ServerList';
import Grid from '@mui/material/Grid';
import Paper from '@mui/material/Paper';

export const Home = (): JSX.Element => {
    return (
        <Grid container spacing={3}>
            <Grid item xs={9}>
                <Paper>
                    <BanList />
                </Paper>
            </Grid>
            <Grid item xs={3}>
                <Paper>
                    <StatsPanel />
                </Paper>
            </Grid>
            <Grid item xs={12}>
                <Paper>
                    <ServerList />
                </Paper>
            </Grid>
        </Grid>
    );
};
