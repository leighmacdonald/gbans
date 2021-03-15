import React from 'react';
import { Grid, Typography } from '@material-ui/core';

export const AdminPeople = (): JSX.Element => {
    return (
        <Grid container>
            <Grid item xs>
                <Typography variant={'h1'}>Manage People</Typography>
            </Grid>
        </Grid>
    );
};
