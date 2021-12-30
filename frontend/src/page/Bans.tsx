import React from 'react';
import { BanList } from '../component/BanList';
import { Grid, Paper } from '@mui/material';

export const Bans = (): JSX.Element => {
    return (
        <Grid container spacing={3}>
            <Grid item xs>
                <Paper>
                    <BanList />
                </Paper>
            </Grid>
        </Grid>
    );
};
