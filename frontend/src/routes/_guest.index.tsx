import AttachMoneyIcon from '@mui/icons-material/AttachMoney';
import ChatIcon from '@mui/icons-material/Chat';
import ElectricBoltIcon from '@mui/icons-material/ElectricBolt';
import EmojiEventsIcon from '@mui/icons-material/EmojiEvents';
import EventIcon from '@mui/icons-material/Event';
import GavelIcon from '@mui/icons-material/Gavel';
import MarkUnreadChatAltIcon from '@mui/icons-material/MarkUnreadChatAlt';
import PieChartIcon from '@mui/icons-material/PieChart';
import StorageIcon from '@mui/icons-material/Storage';
import SupportIcon from '@mui/icons-material/Support';
import VideocamIcon from '@mui/icons-material/Videocam';
import Link from '@mui/material/Link';
import Grid from '@mui/material/Unstable_Grid2';
import { useNavigate, useRouteContext, createFileRoute } from '@tanstack/react-router';
import { PermissionLevel } from '../api';
import { LeftAlignButton } from '../component/LeftAlignButton.tsx';
import { NewsView } from '../component/NewsView';
import RouterLink from '../component/RouterLink.tsx';
import { Title } from '../component/Title.tsx';
import { useAppInfoCtx } from '../contexts/AppInfoCtx.ts';

export const Route = createFileRoute('/_guest/')({
    component: Index
});

function Index() {
    const navigate = useNavigate();
    const { appInfo } = useAppInfoCtx();
    const { profile } = useRouteContext({ from: '/_guest/' });

    return (
        <>
            <Title>Home</Title>
            <Grid container spacing={2}>
                <Grid xs={12} sm={12} md={10}>
                    <NewsView itemsPerPage={3} />
                </Grid>
                <Grid xs={12} sm={12} md={2}>
                    <div>
                        <Grid container spacing={2}>
                            {profile && profile.ban_id == 0 && appInfo.servers_enabled && (
                                <Grid md={12} sm={4}>
                                    <LeftAlignButton
                                        startIcon={<StorageIcon />}
                                        fullWidth
                                        color={'success'}
                                        variant={'contained'}
                                        onClick={async () => {
                                            await navigate({ to: '/servers' });
                                        }}
                                    >
                                        Play Now!
                                    </LeftAlignButton>
                                </Grid>
                            )}
                            {profile && profile.ban_id != 0 && appInfo.reports_enabled && (
                                <Grid md={12} sm={4}>
                                    <LeftAlignButton
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
                                    </LeftAlignButton>
                                </Grid>
                            )}
                            {appInfo.wiki_enabled && (
                                <>
                                    <Grid md={12} sm={4}>
                                        <LeftAlignButton
                                            component={RouterLink}
                                            startIcon={<GavelIcon />}
                                            fullWidth
                                            color={'primary'}
                                            variant={'contained'}
                                            href={`/wiki/Rules`}
                                        >
                                            Rules
                                        </LeftAlignButton>
                                    </Grid>
                                    <Grid md={12} sm={4}>
                                        <LeftAlignButton
                                            component={RouterLink}
                                            startIcon={<EventIcon />}
                                            fullWidth
                                            color={'primary'}
                                            variant={'contained'}
                                            href={'/wiki/Events'}
                                        >
                                            Events
                                        </LeftAlignButton>
                                    </Grid>
                                </>
                            )}
                            {appInfo.patreon_enabled && (
                                <Grid md={12} sm={4}>
                                    <LeftAlignButton
                                        component={RouterLink}
                                        startIcon={<AttachMoneyIcon />}
                                        fullWidth
                                        color={'primary'}
                                        variant={'contained'}
                                        href={`/patreon`}
                                    >
                                        Donate
                                    </LeftAlignButton>
                                </Grid>
                            )}
                            {appInfo.contests_enabled && (
                                <Grid md={12} sm={4}>
                                    <LeftAlignButton
                                        component={RouterLink}
                                        startIcon={<EmojiEventsIcon />}
                                        fullWidth
                                        color={'primary'}
                                        variant={'contained'}
                                        href={`/contests`}
                                    >
                                        Contests
                                    </LeftAlignButton>
                                </Grid>
                            )}
                            {appInfo.chatlogs_enabled && (
                                <Grid md={12} sm={4}>
                                    <LeftAlignButton
                                        component={RouterLink}
                                        startIcon={<ChatIcon />}
                                        fullWidth
                                        color={'primary'}
                                        variant={'contained'}
                                        href={`/chatlogs`}
                                    >
                                        Chat Logs
                                    </LeftAlignButton>
                                </Grid>
                            )}
                            {appInfo.demos_enabled && (
                                <Grid md={12} sm={4}>
                                    <LeftAlignButton
                                        component={RouterLink}
                                        startIcon={<VideocamIcon />}
                                        fullWidth
                                        color={'primary'}
                                        variant={'contained'}
                                        href={`/stv`}
                                    >
                                        SourceTV
                                    </LeftAlignButton>
                                </Grid>
                            )}
                            {appInfo.stats_enabled && (
                                <Grid md={12} sm={4}>
                                    <LeftAlignButton
                                        component={RouterLink}
                                        startIcon={<PieChartIcon />}
                                        fullWidth
                                        color={'primary'}
                                        variant={'contained'}
                                        href={`/stats`}
                                    >
                                        Stats (Beta)
                                    </LeftAlignButton>
                                </Grid>
                            )}
                            {appInfo.speedruns_enabled && profile.permission_level >= PermissionLevel.Moderator && (
                                <Grid md={12} sm={4}>
                                    <LeftAlignButton
                                        component={RouterLink}
                                        startIcon={<ElectricBoltIcon />}
                                        fullWidth
                                        color={'primary'}
                                        variant={'contained'}
                                        href={'/speedruns'}
                                    >
                                        Speedruns
                                    </LeftAlignButton>
                                </Grid>
                            )}
                            {appInfo.discord_enabled && appInfo.link_id != '' && (
                                <Grid md={12} sm={4}>
                                    <LeftAlignButton
                                        component={Link}
                                        startIcon={<MarkUnreadChatAltIcon />}
                                        fullWidth
                                        sx={{ backgroundColor: '#5865F2' }}
                                        variant={'contained'}
                                        href={`https://discord.gg/${appInfo.link_id}`}
                                    >
                                        Join Discord
                                    </LeftAlignButton>
                                </Grid>
                            )}
                        </Grid>
                    </div>
                </Grid>
            </Grid>
        </>
    );
}
