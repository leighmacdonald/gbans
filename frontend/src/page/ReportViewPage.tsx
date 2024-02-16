import { useCallback, useEffect, useState, JSX, useMemo } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import NiceModal from '@ebay/nice-modal-react';
import AccountBalanceIcon from '@mui/icons-material/AccountBalance';
import AutoFixNormalIcon from '@mui/icons-material/AutoFixNormal';
import GavelIcon from '@mui/icons-material/Gavel';
import InfoIcon from '@mui/icons-material/Info';
import PersonSearchIcon from '@mui/icons-material/PersonSearch';
import SendIcon from '@mui/icons-material/Send';
import Avatar from '@mui/material/Avatar';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import ListItemAvatar from '@mui/material/ListItemAvatar';
import ListItemText from '@mui/material/ListItemText';
import MenuItem from '@mui/material/MenuItem';
import Select, { SelectChangeEvent } from '@mui/material/Select';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useTheme } from '@mui/material/styles';
import { isBefore } from 'date-fns';
import {
    apiGetBansSteam,
    apiReportSetState,
    BanReasons,
    BanType,
    PermissionLevel,
    ReportStatus,
    reportStatusColour,
    reportStatusString,
    SteamBanRecord
} from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { Heading } from '../component/Heading';
import { LoadingSpinner } from '../component/LoadingSpinner';
import { ReportViewComponent } from '../component/ReportViewComponent';
import { SteamIDList } from '../component/SteamIDList';
import { ModalBanSteam } from '../component/modal';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { useReport } from '../hooks/useReport.ts';
import { logErr } from '../util/errors';
import { avatarHashToURL } from '../util/text';

export const ReportViewPage = (): JSX.Element => {
    const { report_id } = useParams();
    const theme = useTheme();
    const id = parseInt(report_id || '');
    const [stateAction, setStateAction] = useState(ReportStatus.Opened);
    const [newStateAction, setNewStateAction] = useState(stateAction);
    const { currentUser } = useCurrentUserCtx();
    const [ban, setBan] = useState<SteamBanRecord>();
    const { sendFlash } = useUserFlashCtx();
    const navigate = useNavigate();
    const { data: report } = useReport(id);

    useEffect(() => {
        if (report) {
            setStateAction(report?.report_status);
        }
    }, [report]);

    const handleReportStateChange = (event: SelectChangeEvent<number>) => {
        setNewStateAction(event.target.value as ReportStatus);
    };

    useEffect(() => {
        const abortController = new AbortController();
        apiGetBansSteam(
            { target_id: report?.target_id, desc: true },
            abortController
        ).then((bans) => {
            const active = bans.data.filter((b) =>
                isBefore(new Date(), b.valid_until)
            );
            if (active.length > 0) {
                setBan(active[0]);
            }
        });
        return () => abortController.abort();
    }, [report?.target_id]);

    const onSetReportState = useCallback(() => {
        apiReportSetState(id, newStateAction)
            .then(() => {
                sendFlash(
                    'success',
                    `State changed from ${reportStatusString(
                        report?.report_status ?? ReportStatus.Opened
                    )} => ${reportStatusString(newStateAction)}`
                );
                setStateAction(newStateAction);
            })
            .catch(logErr);
    }, [id, newStateAction, report?.report_status, sendFlash]);

    const renderBan = useMemo(() => {
        if (!ban) {
            return null;
        }

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
    }, [ban, theme.palette.error.main, theme.palette.warning.main]);

    const reportStatusView = useMemo(() => {
        return (
            <ContainerWithHeader
                title={'Report Status'}
                iconLeft={<AccountBalanceIcon />}
            >
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

    return (
        <Grid container spacing={2}>
            <Grid xs={12} md={8}>
                {report && <ReportViewComponent report={report} />}
            </Grid>
            <Grid xs={12} md={4}>
                <Stack spacing={2}>
                    <ContainerWithHeader
                        title={report?.subject.personaname ?? 'Loading'}
                        iconLeft={<PersonSearchIcon />}
                    >
                        <Stack>
                            {!report?.subject.steam_id ? (
                                <LoadingSpinner />
                            ) : (
                                <>
                                    <Avatar
                                        variant={'square'}
                                        alt={report?.subject.personaname}
                                        src={avatarHashToURL(
                                            report?.subject.avatarhash
                                        )}
                                        sx={{
                                            width: '100%',
                                            height: '100%'
                                        }}
                                    />
                                    {renderBan}
                                </>
                            )}
                        </Stack>
                    </ContainerWithHeader>

                    <SteamIDList steam_id={report?.subject.steam_id ?? ''} />

                    {reportStatusView}

                    <ContainerWithHeader
                        title={'Details'}
                        iconLeft={<InfoIcon />}
                    >
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
                                    <Avatar
                                        src={avatarHashToURL(
                                            report?.author.avatarhash
                                        )}
                                    >
                                        <SendIcon />
                                    </Avatar>
                                </ListItemAvatar>
                                <ListItemText
                                    primary={report?.author.personaname}
                                    secondary={'Author'}
                                />
                            </ListItem>
                            {report?.reason && (
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
                                        secondary={BanReasons[report?.reason]}
                                    />
                                </ListItem>
                            )}
                            {report?.reason && report?.reason_text != '' && (
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
                                        primary={'Custom Reason'}
                                        secondary={report?.reason_text}
                                    />
                                </ListItem>
                            )}
                        </List>
                    </ContainerWithHeader>
                    {currentUser.permission_level >=
                        PermissionLevel.Moderator && (
                        <>
                            <ContainerWithHeader
                                title={'Resolve Report'}
                                iconLeft={<AutoFixNormalIcon />}
                            >
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
                                                    value={newStateAction}
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
                                                {report && (
                                                    <Button
                                                        variant={'contained'}
                                                        color={'error'}
                                                        startIcon={
                                                            <GavelIcon />
                                                        }
                                                        onClick={async () => {
                                                            await NiceModal.show(
                                                                ModalBanSteam,
                                                                {
                                                                    reportId:
                                                                        report.report_id,
                                                                    steamId:
                                                                        report
                                                                            ?.subject
                                                                            .steam_id
                                                                }
                                                            );
                                                        }}
                                                    >
                                                        Ban Player
                                                    </Button>
                                                )}
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
                            </ContainerWithHeader>
                        </>
                    )}
                </Stack>
            </Grid>
        </Grid>
    );
};

export default ReportViewPage;
