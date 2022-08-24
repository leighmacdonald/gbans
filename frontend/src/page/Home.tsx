import React from 'react';
import Grid from '@mui/material/Grid';
import Paper from '@mui/material/Paper';
import { NewsView } from '../component/NewsView';
import Stack from '@mui/material/Stack';
import Button from '@mui/material/Button';
import StorageIcon from '@mui/icons-material/Storage';
import GavelIcon from '@mui/icons-material/Gavel';
import EventIcon from '@mui/icons-material/Event';
import SupportIcon from '@mui/icons-material/Support';
import AttachMoneyIcon from '@mui/icons-material/AttachMoney';
import MarkUnreadChatAltIcon from '@mui/icons-material/MarkUnreadChatAlt';
import { useNavigate } from 'react-router-dom';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';

export const Home = (): JSX.Element => {
    const navigate = useNavigate();
    const { currentUser } = useCurrentUserCtx();
    return (
        <Grid container spacing={3} paddingTop={3}>
            <Grid item xs={9}>
                <NewsView itemsPerPage={3} />
            </Grid>
            <Grid item xs={3}>
                <Stack spacing={3}>
                    <Paper elevation={1}>
                        {currentUser.ban_id == 0 ? (
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
                        ) : (
                            <Button
                                startIcon={<SupportIcon />}
                                fullWidth
                                color={'success'}
                                variant={'contained'}
                                onClick={() => {
                                    navigate(`/ban/${currentUser.ban_id}`);
                                }}
                            >
                                Appeal Ban
                            </Button>
                        )}
                    </Paper>
                    <Paper elevation={1}>
                        <Button
                            startIcon={<GavelIcon />}
                            fullWidth
                            color={'primary'}
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
                            color={'primary'}
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
                            startIcon={<AttachMoneyIcon />}
                            fullWidth
                            color={'primary'}
                            variant={'contained'}
                            onClick={() => {
                                navigate('/wiki/Donate');
                            }}
                        >
                            Donate
                        </Button>
                    </Paper>
                    <Paper elevation={1}>
                        <Button
                            startIcon={<MarkUnreadChatAltIcon />}
                            fullWidth
                            color={'primary'}
                            variant={'contained'}
                            onClick={() => {
                                window.open(
                                    'https://discord.gg/uncletopia',
                                    '_blank'
                                );
                            }}
                        >
                            Join Discord
                        </Button>
                    </Paper>
                </Stack>
            </Grid>
        </Grid>
    );
};
