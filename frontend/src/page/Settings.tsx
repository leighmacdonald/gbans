import React from 'react';
import Grid from '@mui/material/Grid';
import { ProfileSettings } from '../component/ProfileSettings';

export const Settings = (): JSX.Element => {
    return (
        <Grid container spacing={2} paddingTop={3}>
            <Grid item xs={6} alignContent={'center'}>
                <ProfileSettings />
            </Grid>
        </Grid>
    );
};
