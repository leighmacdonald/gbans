import React from 'react';
import Grid from '@mui/material/Grid';
import Paper from '@mui/material/Paper';
import { StatsPanel } from '../component/StatsPanel';
import { BanList } from '../component/BanList';
import { ServerList } from '../component/ServerList';

export const Home = (): JSX.Element => {
    return (
        <Grid container spacing={3} paddingTop={3}>
            <Grid item xs={9}>
                <Paper elevation={1}>
                    <BanList />
                </Paper>
            </Grid>
            <Grid item xs={3}>
                <Paper elevation={1}>
                    <StatsPanel />
                </Paper>
            </Grid>
            <Grid item xs={12}>
                <Paper elevation={1}>
                    <ServerList />
                </Paper>
            </Grid>
        </Grid>
    );
};
