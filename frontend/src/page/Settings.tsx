import React from 'react';
import { Grid, Typography } from '@material-ui/core';
import { ProfileSettings } from '../component/ProfileSettings';

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
