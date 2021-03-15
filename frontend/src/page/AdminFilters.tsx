import React from 'react';
import { Grid, Typography } from '@material-ui/core';

export const AdminFilters = (): JSX.Element => {
    return (
        <Grid container>
            <Grid item xs>
                <Typography variant={'h1'}>Filters</Typography>
            </Grid>
        </Grid>
    );
};
