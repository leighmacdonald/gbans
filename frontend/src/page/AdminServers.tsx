import React from 'react';
import { ServerAddForm } from '../component/ServerAddForm';
import { ServerList } from '../component/ServerList';
import { Grid } from '@mui/material';

export const AdminServers = (): JSX.Element => {
    return (
        <Grid container>
            <Grid item xs={8}>
                <ServerList />
            </Grid>
            <Grid item xs={4}>
                <ServerAddForm />
            </Grid>
        </Grid>
    );
};
