import React from 'react';
import { ServerLogView } from '../component/ServerLogView';
import { Grid, Typography } from '@mui/material';

export const AdminServerLog = (): JSX.Element => {
    return (
        <Grid container>
            <Grid item xs={12}>
                <Typography variant={'h1'}>Game Server Logs</Typography>
            </Grid>
            <Grid item xs={12}>
                <ServerLogView />
            </Grid>
        </Grid>
    );
};
