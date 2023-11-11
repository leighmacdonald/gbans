import React, { JSX } from 'react';
import Paper from '@mui/material/Paper';
import Grid from '@mui/material/Unstable_Grid2';
import { STVListView } from '../component/STVListView';

export const STVPage = (): JSX.Element => {
    return (
        <Grid container spacing={2}>
            <Grid xs>
                <Paper elevation={1}>
                    <STVListView />
                </Paper>
            </Grid>
        </Grid>
    );
};
