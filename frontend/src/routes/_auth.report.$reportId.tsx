import { useMemo } from 'react';
import AccountBalanceIcon from '@mui/icons-material/AccountBalance';
import GavelIcon from '@mui/icons-material/Gavel';
import InfoIcon from '@mui/icons-material/Info';
import SendIcon from '@mui/icons-material/Send';
import VolumeOffIcon from '@mui/icons-material/VolumeOff';
import Avatar from '@mui/material/Avatar';
import Grid from '@mui/material/Grid';
import Link from '@mui/material/Link';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemAvatar from '@mui/material/ListItemAvatar';
import ListItemText from '@mui/material/ListItemText';
import Typography from '@mui/material/Typography';
import { useTheme } from '@mui/material/styles';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute, useNavigate, useRouteContext } from '@tanstack/react-router';
import { apiGetBanBySteam, apiGetReport, appealStateString } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { ProfileInfoBox } from '../component/ProfileInfoBox.tsx';
import { ReportModPanel } from '../component/ReportModPanel.tsx';
import { ReportViewComponent } from '../component/ReportViewComponent.tsx';
import RouterLink from '../component/RouterLink.tsx';
import { SteamIDList } from '../component/SteamIDList.tsx';
import { Title } from '../component/Title';
import { BanReasons, BanType } from '../schema/bans.ts';
import { PermissionLevel } from '../schema/people.ts';
import { ReportStatus, reportStatusColour, reportStatusString } from '../schema/report.ts';
import { avatarHashToURL } from '../util/text.tsx';
import { renderDateTime, renderTimeDistance } from '../util/time.ts';

export const Route = createFileRoute('/_auth/report/$reportId')({
    component: ReportView
});

function ReportView() {
    const { reportId } = Route.useParams();
    const theme = useTheme();

    const { hasPermission } = useRouteContext({ from: '/_auth/report/$reportId' });

    const navigate = useNavigate();

    const { data: report, isLoading: isLoadingReport } = useQuery({
        queryKey: ['report', { reportId }],
        queryFn: async () => {
            return await apiGetReport(Number(reportId));
        }
    });

    const { data: ban, isLoading: isLoadingBan } = useQuery({
        queryKey: ['ban', { targetId: report?.target_id }],
        queryFn: async () => {
            if (report?.target_id) {
                return await apiGetBanBySteam(report?.target_id);
            }
        },
        enabled: !isLoadingReport && Boolean(report?.target_id)
    });

    const renderBan = useMemo(() => {
        if (isLoadingBan || !ban || ban.ban_id == 0) {
            return <></>;
        }

        return (
            <ContainerWithHeader
                title={ban.ban_type == BanType.Banned ? 'Banned' : 'Muted'}
                iconLeft={ban.ban_type == BanType.Banned ? <GavelIcon /> : <VolumeOffIcon />}
            >
                <List dense={true}>
                    <ListItem>
                        <ListItemText primary={'Reason'} secondary={BanReasons[ban.reason]} />
                    </ListItem>
                    {ban.reason_text != '' && (
                        <ListItem>
                            <ListItemText primary={'Custom Reason'} secondary={ban.note} />
                        </ListItem>
                    )}
                    <ListItem>
                        <ListItemText primary={'Ban ID'} secondary={ban.ban_id} />
                    </ListItem>
                    <ListItem>
                        <ListItemText primary={'Note'} secondary={ban.note} />
                    </ListItem>
                    <ListItem>
                        <ListItemText primary={'Evasion OK'} secondary={ban.evade_ok ? 'Yes' : 'No'} />
                    </ListItem>
                    <ListItem>
                        <ListItemText primary={'Appeal State'} secondary={appealStateString(ban.appeal_state)} />
                    </ListItem>
                    <ListItem>
                        <ListItemText primary={'Creation Date'} secondary={renderDateTime(ban.created_on)} />
                    </ListItem>
                    <ListItem>
                        <ListItemText
                            primary={'Valid Until Date'}
                            secondary={renderDateTime(ban.valid_until as Date)}
                        />
                    </ListItem>
                    <ListItem>
                        <ListItemText
                            primary={'Expires'}
                            secondary={renderTimeDistance(ban.valid_until as Date, new Date())}
                        />
                    </ListItem>
                    <ListItem>
                        <ListItemText
                            primary={'Author'}
                            secondary={
                                <Link component={RouterLink} to={`/profile/${ban.source_id}`}>
                                    {ban.source_personaname}
                                </Link>
                            }
                        />
                    </ListItem>
                </List>
            </ContainerWithHeader>
        );
    }, [ban, isLoadingBan]);

    const reportStatusView = useMemo(() => {
        return (
            <ContainerWithHeader title={'Report Status'} iconLeft={<AccountBalanceIcon />}>
                <Typography
                    padding={2}
                    variant={'h4'}
                    align={'center'}
                    sx={{
                        color: '#111111',
                        backgroundColor: reportStatusColour(report?.report_status ?? ReportStatus.Any, theme)
                    }}
                >
                    {reportStatusString(report?.report_status ?? ReportStatus.Any)}
                </Typography>
            </ContainerWithHeader>
        );
    }, [report?.report_status, theme]);

    return (
        <Grid container spacing={2}>
            <Title>Report</Title>
            <Grid size={{ xs: 12, md: 8 }}>{report && <ReportViewComponent report={report} />}</Grid>
            <Grid size={{ xs: 12, md: 4 }}>
                <div>
                    <Grid container spacing={2}>
                        <Grid size={{ xs: 6, md: 12 }}>
                            {report?.target_id && <ProfileInfoBox steam_id={report?.target_id} />}
                        </Grid>
                        {renderBan && <Grid size={{ xs: 6, md: 12 }}>{renderBan}</Grid>}
                        <Grid size={{ xs: 6, md: 12 }}>
                            <SteamIDList steam_id={report?.subject.steam_id ?? ''} />
                        </Grid>
                        <Grid size={{ xs: 6, md: 12 }}>{reportStatusView}</Grid>
                        <Grid size={{ xs: 6, md: 12 }}>
                            <ContainerWithHeader title={'Report Details'} iconLeft={<InfoIcon />}>
                                <List sx={{ width: '100%' }}>
                                    <ListItem
                                        sx={{
                                            '&:hover': {
                                                cursor: 'pointer',
                                                backgroundColor: theme.palette.background.paper
                                            }
                                        }}
                                        onClick={async () => {
                                            await navigate({ to: `/profile/${report?.author.steam_id}` });
                                        }}
                                    >
                                        <ListItemAvatar>
                                            <Avatar src={avatarHashToURL(report?.author.avatar_hash)}>
                                                <SendIcon />
                                            </Avatar>
                                        </ListItemAvatar>
                                        <ListItemText primary={report?.author.persona_name} secondary={'Author'} />
                                    </ListItem>
                                    {report?.reason && (
                                        <ListItem
                                            sx={{
                                                '&:hover': {
                                                    cursor: 'pointer',
                                                    backgroundColor: theme.palette.background.paper
                                                }
                                            }}
                                        >
                                            <ListItemText primary={'Reason'} secondary={BanReasons[report?.reason]} />
                                        </ListItem>
                                    )}
                                    {report?.reason && report?.reason_text != '' && (
                                        <ListItem
                                            sx={{
                                                '&:hover': {
                                                    cursor: 'pointer',
                                                    backgroundColor: theme.palette.background.paper
                                                }
                                            }}
                                        >
                                            <ListItemText primary={'Custom Reason'} secondary={report?.reason_text} />
                                        </ListItem>
                                    )}
                                </List>
                            </ContainerWithHeader>
                        </Grid>
                        {hasPermission(PermissionLevel.Moderator) && (
                            <Grid size={{ xs: 6, md: 12 }}>
                                <ReportModPanel reportId={Number(reportId)} />
                            </Grid>
                        )}
                    </Grid>
                </div>
            </Grid>
        </Grid>
    );
}
