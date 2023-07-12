import React, { JSX } from 'react';
import Grid from '@mui/material/Unstable_Grid2';
import Paper from '@mui/material/Paper';
import { STVListVIew } from '../component/STVListVIew';

export const STVPage = (): JSX.Element => {
    return (
        <Grid container paddingTop={3} spacing={2}>
            <Grid xs>
                <Paper elevation={1}>
                    <STVListVIew />
                </Paper>
            </Grid>
        </Grid>
    );
};
