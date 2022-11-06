import React from 'react';
import Grid from '@mui/material/Grid';
import Box from '@mui/material/Box';
import Button from '@mui/material/Button';

export const JoinOrCreateLobby = () => {
    return (
        <Grid container>
            <Grid item xs={6}>
                <Box textAlign="center">
                    <Button color={'success'} variant={'contained'}>
                        Join Lobby
                    </Button>
                </Box>
            </Grid>
            <Grid item xs={6}>
                <Box textAlign="center">
                    <Button color={'success'} variant={'contained'}>
                        Create Lobby
                    </Button>
                </Box>
            </Grid>
        </Grid>
    );
};
