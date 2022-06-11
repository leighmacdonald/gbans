import React from 'react';
import Grid from '@mui/material/Grid';
import Paper from '@mui/material/Paper';
import { BanList } from '../component/BanList';

export const Bans = (): JSX.Element => {
    return (
        <Grid container spacing={3} paddingTop={3}>
            <Grid item xs>
                <Paper elevation={1}>
                    <BanList />
                </Paper>
            </Grid>
        </Grid>
    );
};
