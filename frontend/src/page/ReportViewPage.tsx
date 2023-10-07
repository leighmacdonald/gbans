import React, { useCallback, useEffect, useState, JSX } from 'react';
import Grid from '@mui/material/Unstable_Grid2';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import { ReportComponent } from '../component/ReportComponent';
import { useNavigate, useParams } from 'react-router-dom';
import {
    apiGetBansSteam,
    apiGetReport,
    apiReportSetState,
    BanReasons,
    BanType,
    IAPIBanRecordProfile,
    PermissionLevel,
    ReportStatus,
    reportStatusColour,
    reportStatusString,
    ReportWithAuthor
} from '../api';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import Select, { SelectChangeEvent } from '@mui/material/Select';
import MenuItem from '@mui/material/MenuItem';
import SendIcon from '@mui/icons-material/Send';
import Button from '@mui/material/Button';
import Avatar from '@mui/material/Avatar';
import ListItemAvatar from '@mui/material/ListItemAvatar';
import ListItemText from '@mui/material/ListItemText';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { logErr } from '../util/errors';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { Heading } from '../component/Heading';
import { LoadingSpinner } from '../component/LoadingSpinner';
import { SteamIDList } from '../component/SteamIDList';
import { useTheme } from '@mui/material/styles';
import Typography from '@mui/material/Typography';
import { Nullable } from '../util/types';
import { BanSteamModal } from '../component/BanSteamModal';
import ButtonGroup from '@mui/material/ButtonGroup';
import GavelIcon from '@mui/icons-material/Gavel';

export const ReportViewPage = (): JSX.Element => {
    const { report_id } = useParams();
    const theme = useTheme();
    const id = parseInt(report_id || '');
    const [report, setReport] = useState<ReportWithAuthor>();
    const [stateAction, setStateAction] = useState(ReportStatus.Opened);
    const [banHistory, setBanHistory] = useState<IAPIBanRecordProfile[]>([]);
    const [currentBan, setCurrentBan] =
        useState<Nullable<IAPIBanRecordProfile>>();
    const [banModalOpen, setBanModalOpen] = useState<boolean>(false);
    const { currentUser } = useCurrentUserCtx();
    const { sendFlash } = useUserFlashCtx();
    const navigate = useNavigate();

    const handleReportStateChange = (event: SelectChangeEvent<number>) => {
        setStateAction(event.target.value as ReportStatus);
    };

    useEffect(() => {
        apiGetReport(id)
            .then((response) => {
                setReport(response);
                setStateAction(response.report.report_status);
            })
            .catch((e) => {
                sendFlash(
                    'error',
                    'Permission denied. Only report authors, subjects and mods can view reports'
                );
                logErr(e);
                navigate(`/report`);
                return;
            });
    }, [report_id, setReport, id, sendFlash, navigate]);

    const loadBans = useCallback(() => {
        if (!report?.report.target_id) {
            return;
        }
        apiGetBansSteam({
            limit: 100,
            deleted: true,
            steam_id: report?.report.target_id
        }).then((history) => {
            setBanHistory(history);
            const cur = history.filter((b) => !b.deleted).pop();
            setCurrentBan(cur);
        });
    }, [report?.report.target_id]);

    useEffect(() => {
        loadBans();
    }, [loadBans, report]);

    const onSetReportState = useCallback(() => {
        apiReportSetState(id, stateAction)
            .then(() => {
                sendFlash(
                    'success',
                    `State changed from ${reportStatusString(
                        report?.report.report_status ?? ReportStatus.Opened
                    )} => ${reportStatusString(stateAction)}`
                );
            })
            .catch(logErr);
    }, [id, report?.report.report_status, sendFlash, stateAction]);

    const renderBan = (ban: IAPIBanRecordProfile) => {
        switch (ban.ban_type) {
            case BanType.Banned:
                return (
                    <Heading bgColor={theme.palette.error.main}>Banned</Heading>
                );
            default:
                return (
                    <Heading bgColor={theme.palette.warning.main}>
                        Muted
                    </Heading>
                );
        }
    };

    return (
        <Grid container spacing={2}>
            <Grid xs={12} md={8}>
                {report && (
                    <ReportComponent
                        report={report.report}
                        banHistory={banHistory}
                    />
                )}
            </Grid>
            <Grid xs={12} md={4}>
                <Stack spacing={2}>
                    <Paper elevation={1}>
                        <Stack>
                            {!report?.subject.steam_id ? (
                                <LoadingSpinner />
                            ) : (
                                <>
                                    <Heading>
                                        {report?.subject.personaname}
                                    </Heading>
                                    <Avatar
                                        variant={'square'}
                                        alt={report?.subject.personaname}
                                        src={report?.subject.avatarfull}
                                        sx={{
                                            width: '100%',
                                            height: '100%'
                                        }}
                                    />
                                    {currentBan && renderBan(currentBan)}
                                    <SteamIDList
                                        steam_id={report?.subject.steam_id}
                                    />
                                </>
                            )}
                        </Stack>
                    </Paper>

                    <Paper elevation={1}>
                        <Heading>Report Status</Heading>
                        <Typography
                            padding={2}
                            variant={'h4'}
                            align={'center'}
                            sx={{
                                color: '#111111',
                                backgroundColor: reportStatusColour(
                                    report?.report.report_status ??
                                        ReportStatus.Opened,
                                    theme
                                )
                            }}
                        >
                            {reportStatusString(
                                report?.report.report_status ??
                                    ReportStatus.Opened
                            )}
                        </Typography>
                    </Paper>
                    <Paper elevation={1} sx={{ width: '100%' }}>
                        <Heading>Details</Heading>

                        <List sx={{ width: '100%' }}>
                            <ListItem
                                sx={{
                                    '&:hover': {
                                        cursor: 'pointer',
                                        backgroundColor:
                                            theme.palette.background.paper
                                    }
                                }}
                                onClick={() => {
                                    navigate(
                                        `/profile/${report?.author.steam_id}`
                                    );
                                }}
                            >
                                <ListItemAvatar>
                                    <Avatar src={report?.author.avatar}>
                                        <SendIcon />
                                    </Avatar>
                                </ListItemAvatar>
                                <ListItemText
                                    primary={'Author'}
                                    secondary={report?.author.personaname}
                                />
                            </ListItem>
                            {report?.report.reason && (
                                <ListItem
                                    sx={{
                                        '&:hover': {
                                            cursor: 'pointer',
                                            backgroundColor:
                                                theme.palette.background.paper
                                        }
                                    }}
                                >
                                    <ListItemText
                                        primary={'Reason'}
                                        secondary={
                                            BanReasons[report?.report.reason]
                                        }
                                    />
                                </ListItem>
                            )}
                            {report?.report.reason &&
                                report?.report.reason_text != '' && (
                                    <ListItem
                                        sx={{
                                            '&:hover': {
                                                cursor: 'pointer',
                                                backgroundColor:
                                                    theme.palette.background
                                                        .paper
                                            }
                                        }}
                                    >
                                        <ListItemText
                                            primary={'Custom Reason'}
                                            secondary={
                                                report?.report.reason_text
                                            }
                                        />
                                    </ListItem>
                                )}
                        </List>
                    </Paper>
                    {currentUser.permission_level >=
                        PermissionLevel.Moderator && (
                        <>
                            <Paper elevation={1}>
                                <Heading>Resolve Report</Heading>
                                <List>
                                    <ListItem>
                                        <Stack
                                            sx={{ width: '100%' }}
                                            spacing={2}
                                        >
                                            <FormControl fullWidth>
                                                <InputLabel id="select-label">
                                                    Action
                                                </InputLabel>
                                                <Select<ReportStatus>
                                                    labelId="select-label"
                                                    id="simple-select"
                                                    value={stateAction}
                                                    label="Report State"
                                                    onChange={
                                                        handleReportStateChange
                                                    }
                                                >
                                                    {[
                                                        ReportStatus.Opened,
                                                        ReportStatus.NeedMoreInfo,
                                                        ReportStatus.ClosedWithoutAction,
                                                        ReportStatus.ClosedWithAction
                                                    ].map((status) => (
                                                        <MenuItem
                                                            key={status}
                                                            value={status}
                                                        >
                                                            {reportStatusString(
                                                                status
                                                            )}
                                                        </MenuItem>
                                                    ))}
                                                </Select>
                                            </FormControl>
                                            <ButtonGroup fullWidth>
                                                <Button
                                                    variant={'contained'}
                                                    color={'error'}
                                                    startIcon={<GavelIcon />}
                                                    onClick={() => {
                                                        setBanModalOpen(true);
                                                    }}
                                                >
                                                    Ban Player
                                                </Button>
                                                <Button
                                                    variant={'contained'}
                                                    color={'warning'}
                                                    startIcon={<SendIcon />}
                                                    onClick={onSetReportState}
                                                >
                                                    Set State
                                                </Button>
                                            </ButtonGroup>
                                        </Stack>
                                    </ListItem>
                                </List>
                            </Paper>
                        </>
                    )}
                    {report?.subject.steam_id ? (
                        <BanSteamModal
                            reportId={report.report.report_id}
                            open={banModalOpen}
                            setOpen={setBanModalOpen}
                            steamId={report?.subject.steam_id}
                        />
                    ) : (
                        <></>
                    )}
                </Stack>
            </Grid>
        </Grid>
    );
};
