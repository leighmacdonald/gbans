import React, { JSX } from 'react';
import Paper from '@mui/material/Paper';
import Grid from '@mui/material/Unstable_Grid2';
import { STVListPage } from '../component/STVListPage';

export const STVPage = (): JSX.Element => {
    return (
        <Grid container spacing={2}>
            <Grid xs>
                <Paper elevation={1}>
                    <STVListPage />
                </Paper>
            </Grid>
        </Grid>
    );
};
