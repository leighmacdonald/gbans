import React, { JSX } from 'react';
import Grid from '@mui/material/Unstable_Grid2';
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
import Link from '@mui/material/Link';
import VideocamIcon from '@mui/icons-material/Videocam';
import PieChartIcon from '@mui/icons-material/PieChart';
import ChatIcon from '@mui/icons-material/Chat';

export const Home = (): JSX.Element => {
    const navigate = useNavigate();
    const { currentUser } = useCurrentUserCtx();
    return (
        <Grid container spacing={3}>
            <Grid xs={9}>
                <NewsView itemsPerPage={3} />
            </Grid>
            <Grid xs={3}>
                <Stack spacing={3}>
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

                    <Button
                        startIcon={<ChatIcon />}
                        fullWidth
                        color={'primary'}
                        variant={'contained'}
                        onClick={() => {
                            navigate('/chatlogs');
                        }}
                    >
                        Chat Logs
                    </Button>

                    <Button
                        startIcon={<VideocamIcon />}
                        fullWidth
                        color={'primary'}
                        variant={'contained'}
                        onClick={() => {
                            navigate('/stv');
                        }}
                    >
                        SourceTV
                    </Button>

                    <Button
                        startIcon={<PieChartIcon />}
                        fullWidth
                        color={'primary'}
                        variant={'contained'}
                        onClick={() => {
                            navigate('/stats');
                        }}
                    >
                        Map Stats
                    </Button>

                    {window.gbans.discord_link_id != '' && (
                        <Button
                            component={Link}
                            startIcon={<MarkUnreadChatAltIcon />}
                            fullWidth
                            sx={{ backgroundColor: '#5865F2' }}
                            variant={'contained'}
                            href={`https://discord.gg/${window.gbans.discord_link_id}`}
                        >
                            Join Discord
                        </Button>
                    )}
                </Stack>
            </Grid>
        </Grid>
    );
};
