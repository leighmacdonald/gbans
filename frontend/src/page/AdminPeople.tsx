import React from 'react';
import Grid from '@mui/material/Grid';
import Paper from '@mui/material/Paper';
import Typography from '@mui/material/Typography';
import { PlayerList } from '../component/PlayerList';

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
