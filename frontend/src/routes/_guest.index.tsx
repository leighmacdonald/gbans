import AttachMoneyIcon from '@mui/icons-material/AttachMoney';
import ChatIcon from '@mui/icons-material/Chat';
import EmojiEventsIcon from '@mui/icons-material/EmojiEvents';
import EventIcon from '@mui/icons-material/Event';
import GavelIcon from '@mui/icons-material/Gavel';
import MarkUnreadChatAltIcon from '@mui/icons-material/MarkUnreadChatAlt';
import PieChartIcon from '@mui/icons-material/PieChart';
import StorageIcon from '@mui/icons-material/Storage';
import SupportIcon from '@mui/icons-material/Support';
import VideocamIcon from '@mui/icons-material/Videocam';
import Button from '@mui/material/Button';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Grid from '@mui/material/Unstable_Grid2';
import { useNavigate, useRouteContext, createFileRoute } from '@tanstack/react-router';
import { NewsView } from '../component/NewsView';
import RouterLink from '../component/RouterLink.tsx';

export const Route = createFileRoute('/_guest/')({
    component: Index
});

function Index() {
    const navigate = useNavigate();
    const { profile } = useRouteContext({ from: '/_guest/' });
    return (
        <Grid container spacing={3}>
            <Grid xs={12} sm={12} md={9}>
                <NewsView itemsPerPage={3} />
            </Grid>
            <Grid xs={12} sm={12} md={3}>
                <Stack spacing={3}>
                    {profile.ban_id == 0 ? (
                        <Button
                            startIcon={<StorageIcon />}
                            fullWidth
                            color={'success'}
                            variant={'contained'}
                            onClick={async () => {
                                await navigate({ to: '/servers' });
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
                            onClick={async () => {
                                await navigate({
                                    to: `/ban/${profile.ban_id}`
                                });
                            }}
                        >
                            Appeal Ban
                        </Button>
                    )}

                    <Button
                        component={RouterLink}
                        startIcon={<GavelIcon />}
                        fullWidth
                        color={'primary'}
                        variant={'contained'}
                        to={`/wiki/Rules`}
                    >
                        Rules
                    </Button>

                    <Button
                        component={RouterLink}
                        startIcon={<EventIcon />}
                        fullWidth
                        color={'primary'}
                        variant={'contained'}
                        to={'/wiki/Events'}
                    >
                        Events
                    </Button>

                    <Button
                        component={RouterLink}
                        startIcon={<AttachMoneyIcon />}
                        fullWidth
                        color={'primary'}
                        variant={'contained'}
                        to={`/wiki/Donate`}
                    >
                        Donate
                    </Button>

                    <Button
                        component={RouterLink}
                        startIcon={<EmojiEventsIcon />}
                        fullWidth
                        color={'primary'}
                        variant={'contained'}
                        to={`/contests`}
                    >
                        Contests
                    </Button>

                    <Button
                        component={RouterLink}
                        startIcon={<ChatIcon />}
                        fullWidth
                        color={'primary'}
                        variant={'contained'}
                        to={`/chatlogs`}
                    >
                        Chat Logs
                    </Button>

                    <Button
                        component={RouterLink}
                        startIcon={<VideocamIcon />}
                        fullWidth
                        color={'primary'}
                        variant={'contained'}
                        to={`/stv`}
                    >
                        SourceTV
                    </Button>

                    <Button
                        component={RouterLink}
                        startIcon={<PieChartIcon />}
                        fullWidth
                        color={'primary'}
                        variant={'contained'}
                        to={`/stats`}
                    >
                        Stats (Beta)
                    </Button>

                    {__DISCORD_LINK_ID__ != '' && (
                        <Button
                            component={Link}
                            startIcon={<MarkUnreadChatAltIcon />}
                            fullWidth
                            sx={{ backgroundColor: '#5865F2' }}
                            variant={'contained'}
                            href={`https://discord.gg/${__DISCORD_LINK_ID__}`}
                        >
                            Join Discord
                        </Button>
                    )}
                </Stack>
            </Grid>
        </Grid>
    );
}
