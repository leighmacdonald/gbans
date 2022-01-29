import React from 'react';
import Grid from '@mui/material/Grid';
import { ServerAddForm } from '../component/ServerAddForm';
import { ServerList } from '../component/ServerList';
import Paper from '@mui/material/Paper';

export const AdminServers = (): JSX.Element => {
    return (
        <Grid container spacing={3} paddingTop={3}>
            <Grid item xs={8}>
                <Paper elevation={1}>
                    <ServerList />
                </Paper>
            </Grid>
            <Grid item xs={4}>
                <Paper elevation={1}>
                    <ServerAddForm />
                </Paper>
            </Grid>
        </Grid>
    );
};
