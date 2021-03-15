import React from 'react';
import { Grid, Typography } from '@material-ui/core';

export const ProfileSettings = (): JSX.Element => {
    return (
        <Grid container>
            <Grid item xs>
                <Typography variant={'h3'}>Settings</Typography>
            </Grid>
        </Grid>
    );
};
