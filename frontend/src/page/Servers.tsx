import React from 'react';
import { ServerList } from '../component/ServerList';
import { Grid, Paper } from '@mui/material';

export const Servers = (): JSX.Element => {
    return (
        <Grid container>
            <Grid item xs={12}>
                <Paper>
                    <ServerList />
                </Paper>
            </Grid>
        </Grid>
    );
};
