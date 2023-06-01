import React from 'react';
import Grid from '@mui/material/Grid';
import Typography from '@mui/material/Typography';
import Paper from '@mui/material/Paper';

export const PageNotFound = () => {
    return (
        <Grid container>
            <Grid item xs>
                <Paper elevation={1}>
                    <Typography variant={'h1'}>Page Not Found</Typography>
                </Paper>
            </Grid>
        </Grid>
    );
};
