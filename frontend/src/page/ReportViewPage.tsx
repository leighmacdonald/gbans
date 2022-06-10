import React, { useCallback, useEffect, useState } from 'react';
import Grid from '@mui/material/Grid';
import List from '@mui/material/List';
import ListItem from '@mui/material/ListItem';
import Paper from '@mui/material/Paper';
import ListSubheader from '@mui/material/ListSubheader';
import Stack from '@mui/material/Stack';
import { ReportComponent } from '../component/ReportComponent';
import { useParams } from 'react-router-dom';
import {
    ApiException,
    apiGetReport,
    apiReportSetState,
    PermissionLevel,
    ReportStatus,
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
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import { logErr } from '../util/errors';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';

export const ReportViewPage = (): JSX.Element => {
    const { report_id } = useParams();
    const id = parseInt(report_id || '');
    const [report, setReport] = useState<ReportWithAuthor>();
    const [stateAction, setStateAction] = React.useState(ReportStatus.Opened);
    const { currentUser } = useCurrentUserCtx();
    const { flashes, setFlashes } = useUserFlashCtx();

    const handleReportStateChange = (event: SelectChangeEvent<number>) => {
        setStateAction(event.target.value as ReportStatus);
    };

    useEffect(() => {
        apiGetReport(id)
            .then((r) => {
                if (r) {
                    setReport(r);
                }
            })
            .catch(logErr);
    }, [report_id, setReport, id]);

    const onSetReportState = useCallback(() => {
        apiReportSetState(id, stateAction).catch((error: ApiException) => {
            setFlashes([
                ...flashes,
                {
                    heading: 'Error',
                    level: 'error',
                    message: error.message,
                    closable: true
                }
            ]);
        });
    }, [flashes, id, setFlashes, stateAction]);

    return (
        <Grid container spacing={3} paddingTop={3}>
            <Grid item xs={12}>
                <Box padding={2}>
                    <Typography variant={'h2'}>
                        {report?.report.title}
                    </Typography>
                </Box>
            </Grid>
            <Grid item xs={9}>
                {report && <ReportComponent report={report.report} />}
            </Grid>
            <Grid item xs={3}>
                <Stack spacing={2}>
                    {currentUser.permission_level >=
                        PermissionLevel.Moderator && (
                        <>
                            <Paper elevation={1}>
                                <List
                                    subheader={
                                        <ListSubheader
                                            component="div"
                                            id="nested-list-subheader"
                                        >
                                            Moderation Tools
                                        </ListSubheader>
                                    }
                                >
                                    <ListItem>
                                        <Stack
                                            sx={{ width: '100%' }}
                                            spacing={2}
                                        >
                                            <FormControl fullWidth>
                                                <InputLabel id="select-label">
                                                    Action
                                                </InputLabel>
                                                <Select
                                                    labelId="select-label"
                                                    id="simple-select"
                                                    value={stateAction}
                                                    label="Report State"
                                                    onChange={
                                                        handleReportStateChange
                                                    }
                                                >
                                                    <MenuItem
                                                        value={
                                                            ReportStatus.Opened
                                                        }
                                                    >
                                                        Opened
                                                    </MenuItem>
                                                    <MenuItem
                                                        value={
                                                            ReportStatus.NeedMoreInfo
                                                        }
                                                    >
                                                        Need More Info
                                                    </MenuItem>
                                                    <MenuItem
                                                        value={
                                                            ReportStatus.ClosedWithoutAction
                                                        }
                                                    >
                                                        Closed
                                                    </MenuItem>
                                                    <MenuItem
                                                        value={
                                                            ReportStatus.ClosedWithAction
                                                        }
                                                    >
                                                        Closed (Banned)
                                                    </MenuItem>
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
                            <Paper elevation={1}>
                                <PlayerBanForm />
                            </Paper>
                        </>
                    )}

                    <Paper elevation={1} sx={{ width: '100%' }}>
                        <List
                            sx={{ width: '100%' }}
                            subheader={
                                <ListSubheader
                                    component="div"
                                    id="nested-list-subheader"
                                >
                                    Reporter
                                </ListSubheader>
                            }
                        >
                            <ListItem>
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

                    <Paper elevation={1}>
                        <List
                            subheader={
                                <ListSubheader
                                    component="div"
                                    id="nested-list-subheader"
                                >
                                    Report History
                                </ListSubheader>
                            }
                        />
                    </Paper>
                </Stack>
            </Grid>
        </Grid>
    );
};
