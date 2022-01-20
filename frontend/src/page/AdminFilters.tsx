import React from 'react';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Grid';
import Paper from '@mui/material/Paper';

export const AdminFilters = (): JSX.Element => {
    return (
        <Grid container>
            <Grid item xs>
                <Paper elevation={1}>
                    <Typography variant={'h1'}>Filters</Typography>
                </Paper>
            </Grid>
        </Grid>
    );
};
