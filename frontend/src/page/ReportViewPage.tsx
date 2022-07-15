import React, { useCallback, useEffect, useState } from 'react';
import Grid from '@mui/material/Grid';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import { ReportComponent } from '../component/ReportComponent';
import { useNavigate, useParams } from 'react-router-dom';
import {
    ApiException,
    apiGetBans,
    apiGetReport,
    apiReportSetState,
    BanType,
    IAPIBanRecord,
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
import { PlayerBanForm } from '../component/PlayerBanForm';
import { logErr } from '../util/errors';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { Heading } from '../component/Heading';
import { LoadingSpinner } from '../component/LoadingSpinner';
import { SteamIDList } from '../component/SteamIDList';
import useTheme from '@mui/material/styles/useTheme';
import Typography from '@mui/material/Typography';
import { Nullable } from '../util/types';

export const ReportViewPage = (): JSX.Element => {
    const { report_id } = useParams();
    const theme = useTheme();
    const id = parseInt(report_id || '');
    const [report, setReport] = useState<ReportWithAuthor>();
    const [stateAction, setStateAction] = useState(ReportStatus.Opened);
    const [banHistory, setBanHistory] = useState<IAPIBanRecord[]>([]);
    const [currentBan, setCurrentBan] = useState<Nullable<IAPIBanRecord>>();
    const { currentUser } = useCurrentUserCtx();
    const { sendFlash } = useUserFlashCtx();
    const navigate = useNavigate();

    const handleReportStateChange = (event: SelectChangeEvent<number>) => {
        setStateAction(event.target.value as ReportStatus);
    };

    useEffect(() => {
        apiGetReport(id)
            .then((reportAuthor) => {
                if (reportAuthor) {
                    setReport(reportAuthor);
                    setStateAction(reportAuthor.report.report_status);
                }
            })
            .catch((error: ApiException) => {
                sendFlash(
                    'error',
                    'Permission denied. Only report authors, subjects and mods can view reports'
                );
                logErr(error);
                navigate(`/report`);
            });
    }, [report_id, setReport, id, sendFlash, navigate]);

    useEffect(() => {
        if (!report?.report.reported_id) {
            return;
        }
        apiGetBans({
            limit: 100,
            deleted: true,
            steam_id: report?.report.reported_id
        }).then((history) => {
            setBanHistory(history);
            const cur = history.filter((b) => !b.deleted).pop();
            setCurrentBan(cur);
            console.log(cur);
        });
    }, [report?.report.reported_id]);

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
            .catch((error: ApiException) => {
                sendFlash(
                    'error',
                    `Failed to set report state: ${error.message}`
                );
            });
    }, [id, report?.report.report_status, sendFlash, stateAction]);

    const renderBan = (ban: IAPIBanRecord) => {
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
        <Grid container spacing={3} paddingTop={3}>
            <Grid item xs={12} md={9}>
                {report && (
                    <ReportComponent
                        report={report.report}
                        banHistory={banHistory}
                    />
                )}
            </Grid>
            <Grid item xs={12} md={3}>
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
                        <Heading>Reporter</Heading>
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
                                    primary={report?.author.personaname}
                                    secondary={'Reports: 12'}
                                />
                            </ListItem>
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
                                            <Button
                                                fullWidth
                                                variant={'contained'}
                                                color={'primary'}
                                                endIcon={<SendIcon />}
                                                onClick={onSetReportState}
                                            >
                                                Set Report State
                                            </Button>
                                        </Stack>
                                    </ListItem>
                                </List>
                            </Paper>
                            {report?.subject.steam_id && (
                                <Paper elevation={1}>
                                    <PlayerBanForm
                                        steamId={report?.subject.steam_id}
                                    />
                                </Paper>
                            )}
                        </>
                    )}
                </Stack>
            </Grid>
        </Grid>
    );
};
