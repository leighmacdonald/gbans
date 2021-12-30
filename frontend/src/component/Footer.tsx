import Grid from '@mui/material/Grid';
import React from 'react';
import { Typography } from '@mui/material';

export const Footer = (): JSX.Element => {
    return (
        <Grid
            container
            spacing={3}
            alignItems="center"
            justifyContent="space-evenly"
        >
            <Grid item xs={6}>
                <Typography align={'center'} variant={'body2'}>
                    <a href="https://github.com/leighmacdonald/gbans">gbans</a>
                </Typography>
            </Grid>
        </Grid>
    );
};
