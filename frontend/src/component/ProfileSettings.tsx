import React from 'react';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Grid';

export const ProfileSettings = (): JSX.Element => {
    return (
        <Grid container>
            <Grid item xs>
                <Typography variant={'h3'}>Settings</Typography>
            </Grid>
        </Grid>
    );
};
