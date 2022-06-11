import React from 'react';
import Grid from '@mui/material/Grid';
import Paper from '@mui/material/Paper';
import Typography from '@mui/material/Typography';
import { PlayerList } from '../component/PlayerList';
import Stack from '@mui/material/Stack';

export const AdminPeople = (): JSX.Element => {
    return (
        <Grid container spacing={3} paddingTop={3}>
            <Grid item xs={12}>
                <Paper elevation={1}>
                    <Stack padding={3}>
                        <Typography variant={'h2'}>Known Players</Typography>
                        <PlayerList />
                    </Stack>
                </Paper>
            </Grid>
        </Grid>
    );
};
