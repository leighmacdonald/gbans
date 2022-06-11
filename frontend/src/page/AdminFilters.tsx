import React from 'react';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Grid';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';

export const AdminFilters = (): JSX.Element => {
    return (
        <Grid container spacing={3} paddingTop={3}>
            <Grid item xs>
                <Paper elevation={1}>
                    <Stack spacing={3} padding={3}>
                        <Typography variant={'h1'}>Filters</Typography>
                    </Stack>
                </Paper>
            </Grid>
        </Grid>
    );
};
