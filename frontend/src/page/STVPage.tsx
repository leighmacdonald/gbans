import React from 'react';
import Grid from '@mui/material/Grid';
import Paper from '@mui/material/Paper';
import { STVListVIew } from '../component/STVListVIew';

export const STVPage = (): JSX.Element => {
    return (
        <Grid container paddingTop={3} spacing={2}>
            <Grid item xs>
                <Paper elevation={1}>
                    <STVListVIew />
                </Paper>
            </Grid>
        </Grid>
    );
};
