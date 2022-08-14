import React from 'react';
import Grid from '@mui/material/Grid';
import Paper from '@mui/material/Paper';
import { StatsPanel } from '../component/StatsPanel';
import { NewsView } from '../component/NewsView';
import Stack from '@mui/material/Stack';
import Button from '@mui/material/Button';
import StorageIcon from '@mui/icons-material/Storage';
import GavelIcon from '@mui/icons-material/Gavel';
import EventIcon from '@mui/icons-material/Event';
import MarkUnreadChatAltIcon from '@mui/icons-material/MarkUnreadChatAlt';
import { useNavigate } from 'react-router-dom';

export const Home = (): JSX.Element => {
    const navigate = useNavigate();
    return (
        <Grid container spacing={3} paddingTop={3}>
            <Grid item xs={9}>
                <NewsView itemsPerPage={3} />
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
                        <Button
                            startIcon={<GavelIcon />}
                            fullWidth
                            color={'secondary'}
                            variant={'contained'}
                            onClick={() => {
                                navigate('/wiki/Rules');
                            }}
                        >
                            Rules
                        </Button>
                    </Paper>
                    <Paper elevation={1}>
                        <Button
                            startIcon={<EventIcon />}
                            fullWidth
                            color={'secondary'}
                            variant={'contained'}
                            onClick={() => {
                                navigate('/wiki/Events');
                            }}
                        >
                            Events
                        </Button>
                    </Paper>
                    <Paper elevation={1}>
                        <Button
                            startIcon={<MarkUnreadChatAltIcon />}
                            fullWidth
                            color={'secondary'}
                            variant={'contained'}
                            onClick={() => {
                                window.open('https://discord.gg/uncletopia');
                            }}
                        >
                            Discord
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
