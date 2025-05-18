import { useCallback, useEffect, useMemo, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import AccountBalanceIcon from '@mui/icons-material/AccountBalance';
import AutoFixNormalIcon from '@mui/icons-material/AutoFixNormal';
import GavelIcon from '@mui/icons-material/Gavel';
import InfoIcon from '@mui/icons-material/Info';
import SendIcon from '@mui/icons-material/Send';
import VolumeOffIcon from '@mui/icons-material/VolumeOff';
import Avatar from '@mui/material/Avatar';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import FormControl from '@mui/material/FormControl';
import Grid from '@mui/material/Grid';
import InputLabel from '@mui/material/InputLabel';
import Link from '@mui/material/Link';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemAvatar from '@mui/material/ListItemAvatar';
import ListItemText from '@mui/material/ListItemText';
import MenuItem from '@mui/material/MenuItem';
import Select, { SelectChangeEvent } from '@mui/material/Select';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useTheme } from '@mui/material/styles';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { createFileRoute, useNavigate, useRouteContext } from '@tanstack/react-router';
import { apiGetBanBySteam, apiGetReport, apiReportSetState, appealStateString } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { ProfileInfoBox } from '../component/ProfileInfoBox.tsx';
import { ReportViewComponent } from '../component/ReportViewComponent.tsx';
import RouterLink from '../component/RouterLink.tsx';
import { SteamIDList } from '../component/SteamIDList.tsx';
import { Title } from '../component/Title';
import { ModalBanSteam } from '../component/modal';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { BanReasons, BanType } from '../schema/bans.ts';
import { PermissionLevel } from '../schema/people.ts';
import { ReportStatus, reportStatusColour, ReportStatusEnum, reportStatusString } from '../schema/report.ts';
import { avatarHashToURL } from '../util/text.tsx';
import { renderDateTime, renderTimeDistance } from '../util/time.ts';

export const Route = createFileRoute('/_auth/report/$reportId')({
    component: ReportView
});

function ReportView() {
    const { reportId } = Route.useParams();
    const theme = useTheme();
    const [stateAction, setStateAction] = useState<ReportStatusEnum>(ReportStatus.Opened);
    const [newStateAction, setNewStateAction] = useState<ReportStatusEnum>(stateAction);
    const { hasPermission } = useRouteContext({ from: '/_auth/report/$reportId' });
    const { sendFlash, sendError } = useUserFlashCtx();
    const navigate = useNavigate();
    const queryClient = useQueryClient();

    const { data: report, isLoading: isLoadingReport } = useQuery({
        queryKey: ['report', { reportId }],
        queryFn: async () => {
            return await apiGetReport(Number(reportId));
        }
    });

    useEffect(() => {
        if (!report) {
            return;
        }
        setStateAction(report.report_status);
        setNewStateAction(report.report_status);
    }, [report]);

    const { data: ban, isLoading: isLoadingBan } = useQuery({
        queryKey: ['ban', { targetId: report?.target_id }],
        queryFn: async () => {
            if (report?.target_id) {
                return await apiGetBanBySteam(report?.target_id);
            }
        },
        enabled: !isLoadingReport && Boolean(report?.target_id)
    });

    const handleReportStateChange = (event: SelectChangeEvent<number>) => {
        setNewStateAction(event.target.value as ReportStatusEnum);
    };

    const stateMutation = useMutation({
        mutationKey: ['reportState', { stateAction }],
        mutationFn: async () => {
            return await apiReportSetState(Number(reportId), newStateAction);
        },
        onSuccess: async () => {
            setStateAction(newStateAction);
            sendFlash(
                'success',
                `State changed from ${reportStatusString(
                    report?.report_status ?? ReportStatus.Opened
                )} => ${reportStatusString(newStateAction)}`
            );
        },
        onError: sendError
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
                        <ListItemText primary={'Include Friends'} secondary={ban.include_friends ? 'Yes' : 'No'} />
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
                        backgroundColor: reportStatusColour(stateAction, theme)
                    }}
                >
                    {reportStatusString(stateAction)}
                </Typography>
            </ContainerWithHeader>
        );
    }, [stateAction, theme]);

    const onBan = useCallback(async () => {
        if (!report) {
            return;
        }

        try {
            const banRecord = await NiceModal.show(ModalBanSteam, {
                reportId: report.report_id,
                steamId: report.subject.steam_id
            });
            queryClient.setQueryData(['ban', { targetId: report?.target_id }], banRecord);
            setStateAction(ReportStatus.ClosedWithAction);
            setNewStateAction(ReportStatus.ClosedWithAction);
        } catch (e) {
            sendFlash('error', `Failed to ban: ${e}`);
        }
    }, [queryClient, report, sendFlash]);

    const resolveView = useMemo(() => {
        return (
            <ContainerWithHeader title={'Resolve Report'} iconLeft={<AutoFixNormalIcon />}>
                <List>
                    <ListItem>
                        <Stack sx={{ width: '100%' }} spacing={2}>
                            <FormControl fullWidth>
                                <InputLabel id="select-label">Action</InputLabel>
                                <Select
                                    labelId="select-label"
                                    id="simple-select"
                                    value={newStateAction}
                                    label="Report State"
                                    onChange={handleReportStateChange}
                                >
                                    {[
                                        ReportStatus.Opened,
                                        ReportStatus.NeedMoreInfo,
                                        ReportStatus.ClosedWithoutAction,
                                        ReportStatus.ClosedWithAction
                                    ].map((status) => (
                                        <MenuItem key={status} value={status}>
                                            {reportStatusString(status)}
                                        </MenuItem>
                                    ))}
                                </Select>
                            </FormControl>
                            <ButtonGroup fullWidth>
                                {report && (
                                    <Button
                                        variant={'contained'}
                                        color={'error'}
                                        startIcon={<GavelIcon />}
                                        onClick={onBan}
                                        disabled={(ban?.ban_id ?? 0) > 0}
                                    >
                                        Ban Player
                                    </Button>
                                )}
                                <Button
                                    variant={'contained'}
                                    color={'warning'}
                                    disabled={stateAction == newStateAction}
                                    startIcon={<SendIcon />}
                                    onClick={() => {
                                        stateMutation.mutate();
                                    }}
                                >
                                    Set State
                                </Button>
                            </ButtonGroup>
                        </Stack>
                    </ListItem>
                </List>
            </ContainerWithHeader>
        );
    }, [ban?.ban_id, newStateAction, onBan, report, stateAction, stateMutation]);

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
                                            <Avatar src={avatarHashToURL(report?.author.avatarhash)}>
                                                <SendIcon />
                                            </Avatar>
                                        </ListItemAvatar>
                                        <ListItemText primary={report?.author.personaname} secondary={'Author'} />
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
                        <Grid size={{ xs: 6, md: 12 }}>{hasPermission(PermissionLevel.Moderator) && resolveView}</Grid>
                    </Grid>
                </div>
            </Grid>
        </Grid>
    );
}
