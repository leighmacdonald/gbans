import React from 'react';
import Grid from '@mui/material/Grid';
import Typography from '@mui/material/Typography';
import { ProfileSettings } from '../component/ProfileSettings';
import Paper from '@mui/material/Paper';

export const Settings = (): JSX.Element => {
    return (
        <Grid container>
            <Grid item xs>
                <Paper elevation={1}>
                    <Typography variant={'h1'}>STV Recordings</Typography>
                </Paper>
            </Grid>
            <Grid item xs>
                <Paper elevation={1}>
                    <ProfileSettings />
                </Paper>
            </Grid>
        </Grid>
    );
};
