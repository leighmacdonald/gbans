import React from 'react';
import { Grid, Typography } from '@mui/material';

export const ProfileSettings = (): JSX.Element => {
    return (
        <Grid container>
            <Grid item xs>
                <Typography variant={'h3'}>Settings</Typography>
            </Grid>
        </Grid>
    );
};
