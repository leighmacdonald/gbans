import React from 'react';
import Grid from '@mui/material/Grid';
import Paper from '@mui/material/Paper';
import { STVListVIew } from '../component/STVListVIew';

export const STVPage = (): JSX.Element => {
    return (
        <Grid container>
            <Grid item xs>
                <Paper elevation={1}>
                    <STVListVIew />
                </Paper>
            </Grid>
        </Grid>
    );
};
