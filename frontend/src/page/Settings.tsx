import React from 'react';
import { ProfileSettings } from '../component/ProfileSettings';
import { Grid, Typography } from '@mui/material';

export const Settings = (): JSX.Element => {
    return (
        <Grid container>
            <Grid item xs>
                <Typography variant={'h1'}>User Settings</Typography>
            </Grid>
            <Grid item xs>
                <ProfileSettings />
            </Grid>
        </Grid>
    );
};
