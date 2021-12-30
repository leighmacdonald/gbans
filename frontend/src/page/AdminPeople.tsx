import React from 'react';
import { PlayerList } from '../component/PlayerList';
import { Grid, Paper, Typography } from '@mui/material';

export const AdminPeople = (): JSX.Element => {
    return (
        <Grid container spacing={3}>
            <Grid item xs={12}>
                <Paper>
                    <Typography variant={'h2'}>Known Players</Typography>
                    <PlayerList />
                </Paper>
            </Grid>
        </Grid>
    );
};
