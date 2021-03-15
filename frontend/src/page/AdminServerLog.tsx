import React from 'react';
import { ServerLogView } from '../component/ServerLogView';
import { Grid, Typography } from '@material-ui/core';

export const AdminServerLog = (): JSX.Element => {
    return (
        <Grid container>
            <Grid item xs>
                <Typography variant={'h1'}>Game Server Logs</Typography>
            </Grid>
            <Grid item xs>
                <ServerLogView />
            </Grid>
        </Grid>
    );
};
