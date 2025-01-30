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
import Stack from '@mui/material/Stack';
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
            <Grid container spacing={3}>
                <Grid xs={12} sm={12} md={10}>
                    <NewsView itemsPerPage={3} />
                </Grid>
                <Grid xs={12} sm={12} md={2}>
                    <Stack spacing={3}>
                        {profile && profile.ban_id == 0 && appInfo.servers_enabled && (
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
                        )}
                        {profile && profile.ban_id != 0 && appInfo.reports_enabled && (
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
                        )}
                        {appInfo.wiki_enabled && (
                            <>
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
                            </>
                        )}
                        {appInfo.patreon_enabled && (
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
                        )}
                        {appInfo.contests_enabled && (
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
                        )}
                        {appInfo.chatlogs_enabled && (
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
                        )}
                        {appInfo.demos_enabled && (
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
                        )}
                        {appInfo.stats_enabled && (
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
                        )}
                        {appInfo.speedruns_enabled && profile.permission_level >= PermissionLevel.Moderator && (
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
                        )}
                        {appInfo.discord_enabled && appInfo.link_id != '' && (
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
                        )}
                    </Stack>
                </Grid>
            </Grid>
        </>
    );
}
