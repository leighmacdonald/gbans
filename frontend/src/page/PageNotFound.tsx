import React from 'react';
import { handleOnLogout } from '../util/api';
import { Grid, Typography } from '@material-ui/core';

export const PageNotFound = (): JSX.Element => {
    handleOnLogout();
    return (
        <Grid container>
            <Grid item xs>
                <Typography variant={'h1'}>Not Found</Typography>
            </Grid>
        </Grid>
    );
};
