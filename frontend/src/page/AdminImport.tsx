import React from 'react';
import { Grid, Typography } from '@mui/material';

export const AdminImport = (): JSX.Element => {
    return (
        <Grid container>
            <Grid item xs>
                <Typography variant={'h1'}>
                    Import Bans & Block Lists
                </Typography>
            </Grid>
        </Grid>
    );
};
