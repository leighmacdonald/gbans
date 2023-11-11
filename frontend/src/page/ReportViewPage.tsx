import React, { useCallback, useEffect, useState, JSX } from 'react';
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
import {
    apiGetReport,
    apiReportSetState,
    BanReasons,
    PermissionLevel,
    ReportStatus,
    reportStatusColour,
    reportStatusString,
    ReportWithAuthor
} from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { LoadingSpinner } from '../component/LoadingSpinner';
import { ReportComponent } from '../component/ReportComponent';
import { SteamIDList } from '../component/SteamIDList';
import { ModalBanSteam } from '../component/modal';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { logErr } from '../util/errors';

export const ReportViewPage = (): JSX.Element => {
    const { report_id } = useParams();
    const theme = useTheme();
    const id = parseInt(report_id || '');
    const [report, setReport] = useState<ReportWithAuthor>();
    const [stateAction, setStateAction] = useState(ReportStatus.Opened);
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
                setStateAction(response.report_status);
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

    const onSetReportState = useCallback(() => {
        apiReportSetState(id, stateAction)
            .then(() => {
                sendFlash(
                    'success',
                    `State changed from ${reportStatusString(
                        report?.report_status ?? ReportStatus.Opened
                    )} => ${reportStatusString(stateAction)}`
                );
            })
            .catch(logErr);
    }, [id, report?.report_status, sendFlash, stateAction]);

    // const renderBan = (ban: SteamBanRecord) => {
    //     switch (ban.ban_type) {
    //         case BanType.Banned:
    //             return (
    //                 <Heading bgColor={theme.palette.error.main}>Banned</Heading>
    //             );
    //         default:
    //             return (
    //                 <Heading bgColor={theme.palette.warning.main}>
    //                     Muted
    //                 </Heading>
    //             );
    //     }
    // };

    return (
        <Grid container spacing={2}>
            <Grid xs={12} md={8}>
                {report && <ReportComponent report={report} />}
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
                                        src={report?.subject.avatarfull}
                                        sx={{
                                            width: '100%',
                                            height: '100%'
                                        }}
                                    />
                                    {/*currentBan && renderBan(currentBan)*/}
                                </>
                            )}
                        </Stack>
                    </ContainerWithHeader>

                    <SteamIDList steam_id={report?.subject.steam_id ?? ''} />

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
                                backgroundColor: reportStatusColour(
                                    report?.report_status ??
                                        ReportStatus.Opened,
                                    theme
                                )
                            }}
                        >
                            {reportStatusString(
                                report?.report_status ?? ReportStatus.Opened
                            )}
                        </Typography>
                    </ContainerWithHeader>
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
                                    <Avatar src={report?.author.avatar}>
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
