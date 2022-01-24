import React from 'react';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Grid';

export const AdminServerLog = (): JSX.Element => {
    return (
        <Grid container>
            <Grid item xs={12}>
                <Typography variant={'h1'}>Game Server Logs</Typography>
            </Grid>
            <Grid item xs={12} />
        </Grid>
    );
};
