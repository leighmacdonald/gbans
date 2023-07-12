import React from 'react';
import Grid from '@mui/material/Unstable_Grid2';
import { ProfileSettings } from '../component/ProfileSettings';

export const Settings = () => {
    return (
        <Grid container spacing={2} paddingTop={3}>
            <Grid xs={6} alignContent={'center'}>
                <ProfileSettings />
            </Grid>
        </Grid>
    );
};
