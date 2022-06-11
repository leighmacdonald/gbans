import React from 'react';
import Grid from '@mui/material/Grid';
import Paper from '@mui/material/Paper';
import { StatsPanel } from '../component/StatsPanel';
import { NewsView } from '../component/NewsView';
import Stack from '@mui/material/Stack';
import Button from '@mui/material/Button';
import StorageIcon from '@mui/icons-material/Storage';
import { useNavigate } from 'react-router-dom';

export const Home = (): JSX.Element => {
    const navigate = useNavigate();
    return (
        <Grid container spacing={3} paddingTop={3}>
            <Grid item xs={9}>
                <NewsView />
            </Grid>
            <Grid item xs={3}>
                <Stack spacing={3}>
                    <Paper elevation={1}>
                        <Button
                            startIcon={<StorageIcon />}
                            fullWidth
                            color={'success'}
                            variant={'contained'}
                            onClick={() => {
                                navigate('/servers');
                            }}
                        >
                            Play Now!
                        </Button>
                    </Paper>
                    <Paper elevation={1}>
                        <StatsPanel />
                    </Paper>
                </Stack>
            </Grid>
        </Grid>
    );
};
