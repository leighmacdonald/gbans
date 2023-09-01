import React, { useCallback, useEffect, useState } from 'react';
import Grid from '@mui/material/Unstable_Grid2';
import Paper from '@mui/material/Paper';
import Typography from '@mui/material/Typography';
import Select, { SelectChangeEvent } from '@mui/material/Select';
import Stack from '@mui/material/Stack';
import { DesktopDatePicker } from '@mui/x-date-pickers/DesktopDatePicker';
import MenuItem from '@mui/material/MenuItem';
import {
    apiGetMessages,
    apiGetServers,
    MessageQuery,
    PermissionLevel,
    PersonMessage,
    Server,
    ServerSimple,
    sessionKeyReportPersonMessageIdName,
    sessionKeyReportSteamID
} from '../api';
import { steamIdQueryValue } from '../util/text';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import { Heading } from '../component/Heading';
import { LazyTable } from '../component/LazyTable';
import { logErr } from '../util/errors';
import { Order, RowsPerPage } from '../component/DataTable';
import { formatISO9075 } from 'date-fns/fp';
import { Divider, IconButton, TablePagination } from '@mui/material';
import { useTimer } from 'react-timer-hook';
import ChatIcon from '@mui/icons-material/Chat';
import FilterAltIcon from '@mui/icons-material/FilterAlt';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import Box from '@mui/material/Box';
import { DelayedTextInput } from '../component/DelayedTextInput';
import { parseISO } from 'date-fns';
import FlagIcon from '@mui/icons-material/Flag';
import SettingsSuggestIcon from '@mui/icons-material/SettingsSuggest';
import Menu from '@mui/material/Menu';
import { useNavigate } from 'react-router-dom';
import ListItemIcon from '@mui/material/ListItemIcon';
import ReportIcon from '@mui/icons-material/Report';
import ReportGmailerrorredIcon from '@mui/icons-material/ReportGmailerrorred';
import ListItemText from '@mui/material/ListItemText';
import HistoryIcon from '@mui/icons-material/History';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import stc from 'string-to-color';
import { PersonCell } from '../component/PersonCell';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import CircularProgress from '@mui/material/CircularProgress';

const anyServer: Server = {
    server_name: 'Any',
    server_id: 0,
    server_name_long: 'Any',
    address: '',
    port: 27015,
    longitude: 0.0,
    latitude: 0.0,
    is_enabled: true,
    cc: '',
    default_map: '',
    password: '',
    rcon: '',
    players_max: 24,
    region: '',
    reserved_slots: 8,
    colour: '',
    updated_on: new Date(),
    created_on: new Date()
};

const anyServerSimple: ServerSimple = {
    server_name: 'Any',
    server_id: 0,
    server_name_long: 'Any',
    colour: ''
};

interface ChatQueryState<T> {
    startDate: Date | null | string;
    endDate: Date | null | string;
    steamId: string;
    nameQuery: string;
    messageQuery: string;
    sortOrder: Order;
    sortColumn: keyof T;
    page: number;
    rowPerPageCount: number;
    selectedServer: number;
}

const localStorageKey = 'chat_query_state';

const loadState = () => {
    let config: ChatQueryState<PersonMessage> = {
        startDate: null,
        endDate: null,
        sortOrder: 'desc',
        sortColumn: 'person_message_id',
        selectedServer: anyServer.server_id,
        rowPerPageCount: RowsPerPage.Fifty,
        nameQuery: '',
        messageQuery: '',
        steamId: '',
        page: 0
    };
    const item = localStorage.getItem(localStorageKey);
    if (item) {
        config = JSON.parse(item);
        if (config.startDate) {
            config.startDate = parseISO(config.startDate as string);
        }
        if (config.endDate) {
            config.endDate = parseISO(config.endDate as string);
        }
    }
    return config;
};

export const ChatLogPage = () => {
    const init = loadState();
    const [startDate, setStartDate] = useState<Date | null | string>(
        init.startDate
    );
    const [endDate, setEndDate] = useState<Date | null | string>(init.endDate);
    const [steamId, setSteamId] = useState<string>(init.steamId);
    const [nameQuery, setNameQuery] = useState<string>(init.nameQuery);
    const [messageQuery, setMessageQuery] = useState<string>(init.messageQuery);
    const [sortOrder, setSortOrder] = useState<Order>(init.sortOrder);
    const [sortColumn, setSortColumn] = useState<keyof PersonMessage>(
        init.sortColumn
    );
    const [servers, setServers] = useState<ServerSimple[]>([]);
    const [rows, setRows] = useState<PersonMessage[]>([]);
    const [page, setPage] = useState(init.page);
    const [rowPerPageCount, setRowPerPageCount] = useState<number>(
        init.rowPerPageCount
    );
    const [refreshTime, setRefreshTime] = useState<number>(0);
    const [totalRows, setTotalRows] = useState<number>(0);
    //const [pageCount, setPageCount] = useState<number>(0);
    const [loading, setLoading] = useState(false);
    const [nameValue, setNameValue] = useState<string>(init.nameQuery);
    const [steamIDValue, setSteamIDValue] = useState<string>(init.steamId);
    const [messageValue, setMessageValue] = useState<string>(init.messageQuery);

    const [selectedServer, setSelectedServer] = useState<number>(
        init.selectedServer ?? ''
    );
    const { currentUser } = useCurrentUserCtx();

    const curTime = new Date();
    curTime.setSeconds(curTime.getSeconds() + refreshTime);

    const { isRunning, restart } = useTimer({
        expiryTimestamp: curTime,
        autoStart: false
    });

    const saveState = useCallback(() => {
        localStorage.setItem(
            localStorageKey,
            JSON.stringify({
                endDate,
                steamId,
                messageQuery,
                nameQuery,
                page,
                rowPerPageCount,
                selectedServer,
                sortColumn,
                sortOrder,
                startDate
            } as ChatQueryState<PersonMessage>)
        );
    }, [
        endDate,
        messageQuery,
        nameQuery,
        page,
        rowPerPageCount,
        selectedServer,
        sortColumn,
        sortOrder,
        startDate,
        steamId
    ]);

    useEffect(() => {
        apiGetServers().then((resp) => {
            if (!resp.status || !resp.result) {
                return;
            }
            setServers([
                anyServerSimple,
                ...resp.result.sort((a, b) => {
                    return a.server_name.localeCompare(b.server_name);
                })
            ]);
        });
    }, []);

    const restartTimer = useCallback(() => {
        if (refreshTime <= 0) {
            return;
        }
        const newTime = new Date();
        newTime.setSeconds(newTime.getSeconds() + refreshTime);
        restart(newTime, true);
    }, [refreshTime, restart]);

    useEffect(() => {
        if (isRunning) {
            // wait for timer to exec
            return;
        }
        const opts: MessageQuery = {};
        if (selectedServer > 0) {
            opts.server_id = selectedServer;
        }

        opts.persona_name = nameQuery;
        opts.query = messageQuery;
        opts.steam_id = steamId;
        opts.sent_after = (startDate as Date) ?? undefined;
        opts.sent_before = (endDate as Date) ?? undefined;
        opts.limit = rowPerPageCount;
        opts.offset = page * rowPerPageCount;
        opts.order_by = sortColumn;
        opts.desc = sortOrder == 'desc';
        setLoading(true);
        apiGetMessages(opts)
            .then((resp) => {
                if (resp.result) {
                    setRows(resp.result.messages || []);
                    setTotalRows(resp.result.count);
                    if (page * rowPerPageCount > resp.result.count) {
                        setPage(0);
                    }
                }
            })
            .catch((e) => {
                logErr(e);
            })
            .finally(() => {
                setLoading(false);
            });
        saveState();
        restartTimer();
    }, [
        endDate,
        messageQuery,
        nameQuery,
        page,
        rowPerPageCount,
        selectedServer,
        sortColumn,
        sortOrder,
        startDate,
        steamId,
        isRunning,
        restart,
        restartTimer,
        saveState
    ]);

    const reset = () => {
        setNameQuery('');
        setNameValue('');
        setSteamId('');
        setSteamIDValue('');
        setSelectedServer(anyServer.server_id);
        setStartDate(null);
        setEndDate(null);
        setPage(0);
        setRefreshTime(0);
        setSortColumn('person_message_id');
        setSortOrder('desc');
        setMessageValue('');
        setMessageQuery('');
    };

    return (
        <Grid container spacing={2}>
            <Grid xs={12}>
                <Paper elevation={1}>
                    <Stack>
                        <Heading iconLeft={<FilterAltIcon />}>
                            Chat Filters
                        </Heading>

                        <Grid
                            container
                            padding={2}
                            spacing={2}
                            justifyContent={'center'}
                            alignItems={'center'}
                        >
                            <Grid xs={6} md={3}>
                                <DelayedTextInput
                                    value={nameValue}
                                    setValue={setNameValue}
                                    placeholder={'Name'}
                                    onChange={(value) => {
                                        setNameQuery(value);
                                    }}
                                />
                            </Grid>
                            <Grid xs={6} md={3}>
                                <DelayedTextInput
                                    value={steamIDValue}
                                    setValue={setSteamIDValue}
                                    placeholder={'Steam ID'}
                                    onChange={(value) => {
                                        setSteamId(value);
                                    }}
                                />
                            </Grid>
                            <Grid xs={6} md={3}>
                                <DelayedTextInput
                                    value={messageValue}
                                    setValue={setMessageValue}
                                    placeholder={'Message'}
                                    onChange={(value) => {
                                        setMessageQuery(value);
                                    }}
                                />
                            </Grid>
                            <Grid xs={6} md={3}>
                                <Select<number>
                                    fullWidth
                                    value={
                                        servers.length > 0 ? selectedServer : ''
                                    }
                                    onChange={(event) => {
                                        servers
                                            .filter(
                                                (s) =>
                                                    s.server_id ==
                                                    event.target.value
                                            )
                                            .map((s) =>
                                                setSelectedServer(s.server_id)
                                            );
                                    }}
                                    label={'Server'}
                                >
                                    {servers.map((server) => {
                                        return (
                                            <MenuItem
                                                value={server.server_id}
                                                key={server.server_id}
                                            >
                                                {server.server_name}
                                            </MenuItem>
                                        );
                                    })}
                                </Select>
                            </Grid>

                            <Grid xs={6} md={3}>
                                <DesktopDatePicker
                                    sx={{ width: '100%' }}
                                    label="Date Start"
                                    format={'MM/dd/yyyy'}
                                    value={startDate as Date}
                                    onChange={(newValue: Date | null) => {
                                        setStartDate(newValue);
                                    }}
                                />
                            </Grid>
                            <Grid xs={6} md={3}>
                                <DesktopDatePicker
                                    sx={{ width: '100%' }}
                                    label="Date End"
                                    format="MM/dd/yyyy"
                                    value={endDate as Date}
                                    onChange={(newValue: Date | null) => {
                                        setEndDate(newValue);
                                    }}
                                />
                            </Grid>
                            <Grid xs md={3} mdOffset="auto">
                                <ButtonGroup size={'large'} fullWidth>
                                    <Button
                                        variant={'contained'}
                                        color={'info'}
                                        onClick={reset}
                                    >
                                        Reset
                                    </Button>
                                </ButtonGroup>
                            </Grid>
                        </Grid>
                    </Stack>
                </Paper>
            </Grid>

            <Grid
                xs={12}
                container
                justifyContent="space-between"
                alignItems="center"
                flexDirection={{ xs: 'column', sm: 'row' }}
            >
                <Grid xs={3}>
                    {currentUser.permission_level >=
                        PermissionLevel.Moderator && (
                        <Box sx={{ width: 120 }}>
                            <FormControl fullWidth>
                                <InputLabel id="auto-refresh-label">
                                    Auto-Refresh
                                </InputLabel>
                                <Select<number>
                                    labelId="auto-refresh-label"
                                    id="auto-refresh"
                                    label="Auto Refresh"
                                    value={refreshTime ?? ''}
                                    onChange={(
                                        event: SelectChangeEvent<number>
                                    ) => {
                                        setRefreshTime(
                                            event.target.value as number
                                        );

                                        restartTimer();
                                    }}
                                >
                                    <MenuItem value={0}>Off</MenuItem>
                                    <MenuItem value={10}>5s</MenuItem>
                                    <MenuItem value={15}>15s</MenuItem>
                                    <MenuItem value={30}>30s</MenuItem>
                                    <MenuItem value={60}>60s</MenuItem>
                                </Select>
                            </FormControl>
                        </Box>
                    )}
                </Grid>
                <Grid xs={'auto'}>
                    <TablePagination
                        SelectProps={{
                            disabled: loading
                        }}
                        backIconButtonProps={
                            loading
                                ? {
                                      disabled: loading
                                  }
                                : undefined
                        }
                        nextIconButtonProps={
                            loading
                                ? {
                                      disabled: loading
                                  }
                                : undefined
                        }
                        variant={'head'}
                        page={page}
                        count={totalRows}
                        showFirstButton
                        showLastButton
                        rowsPerPage={rowPerPageCount}
                        onRowsPerPageChange={(
                            event: React.ChangeEvent<
                                HTMLInputElement | HTMLTextAreaElement
                            >
                        ) => {
                            setRowPerPageCount(
                                parseInt(event.target.value, 10)
                            );
                            setPage(0);
                        }}
                        onPageChange={(_, newPage) => {
                            setPage(newPage);
                        }}
                    />
                </Grid>
            </Grid>

            <Grid xs={12}>
                <ContainerWithHeader
                    iconLeft={
                        loading ? (
                            <CircularProgress color="inherit" size={20} />
                        ) : (
                            <ChatIcon />
                        )
                    }
                    title={'Chat Logs'}
                >
                    <LazyTable<PersonMessage>
                        sortOrder={sortOrder}
                        sortColumn={sortColumn}
                        onSortColumnChanged={async (column) => {
                            setSortColumn(column);
                        }}
                        onSortOrderChanged={async (direction) => {
                            setSortOrder(direction);
                        }}
                        columns={[
                            {
                                label: 'Server',
                                tooltip: 'Server',
                                sortKey: 'server_id',
                                align: 'center',
                                width: 100,
                                onClick: (o) => {
                                    setSelectedServer(o.server_id);
                                },
                                queryValue: (o) =>
                                    `${o.server_id} + ${o.server_name}`,
                                renderer: (row) => (
                                    <Button
                                        fullWidth
                                        variant={'text'}
                                        sx={{
                                            color: stc(row.server_name)
                                        }}
                                    >
                                        {row.server_name}
                                    </Button>
                                )
                            },
                            {
                                label: 'Created',
                                tooltip: 'Time the message was sent',
                                sortKey: 'created_on',
                                sortType: 'date',
                                sortable: false,
                                align: 'center',
                                width: 180,
                                queryValue: (o) =>
                                    steamIdQueryValue(o.steam_id),
                                renderer: (row) => (
                                    <Typography variant={'body1'}>
                                        {`${formatISO9075(row.created_on)}`}
                                    </Typography>
                                )
                            },
                            {
                                label: 'Name',
                                tooltip: 'Persona Name',
                                sortKey: 'persona_name',
                                width: 250,
                                align: 'left',
                                onClick: (o) => {
                                    setSteamId(o.steam_id);
                                    setSteamIDValue(o.steam_id);
                                },
                                queryValue: (o) => `${o.persona_name}`,
                                renderer: (row) => (
                                    <PersonCell
                                        steam_id={row.steam_id}
                                        avatar={`https://avatars.akamai.steamstatic.com/${row.avatar_hash}.jpg`}
                                        personaname={row.persona_name}
                                        onClick={() => {
                                            setSteamId(row.steam_id);
                                        }}
                                    />
                                )
                            },
                            {
                                label: 'Message',
                                tooltip: 'Message',
                                sortKey: 'body',
                                align: 'left',
                                queryValue: (o) => o.body,
                                renderer: (row) => {
                                    return (
                                        <Grid container>
                                            <Grid xs padding={1}>
                                                <Typography variant={'body1'}>
                                                    {row.body}
                                                </Typography>
                                            </Grid>

                                            {row.auto_filter_flagged && (
                                                <Grid
                                                    xs={'auto'}
                                                    padding={1}
                                                    display="flex"
                                                    justifyContent="center"
                                                    alignItems="center"
                                                >
                                                    <>
                                                        <FlagIcon
                                                            color={'error'}
                                                            fontSize="small"
                                                        />
                                                    </>
                                                </Grid>
                                            )}
                                            <Grid
                                                xs={'auto'}
                                                display="flex"
                                                justifyContent="center"
                                                alignItems="center"
                                            >
                                                <ChatContextMenu
                                                    flagged={
                                                        row.auto_filter_flagged
                                                    }
                                                    steamId={row.steam_id}
                                                    person_message_id={
                                                        row.person_message_id
                                                    }
                                                />
                                            </Grid>
                                        </Grid>
                                    );
                                }
                            }
                        ]}
                        rows={rows}
                    />
                </ContainerWithHeader>
            </Grid>
        </Grid>
    );
};

interface ChatContextMenuProps {
    person_message_id: number;
    flagged: boolean;
    steamId: string;
}

const ChatContextMenu = ({
    person_message_id,
    flagged,
    steamId
}: ChatContextMenuProps) => {
    const navigate = useNavigate();

    const [anchorEl, setAnchorEl] = React.useState<null | HTMLElement>(null);
    const open = Boolean(anchorEl);
    const handleClick = (event: React.MouseEvent<HTMLElement>) => {
        setAnchorEl(event.currentTarget);
    };
    const handleClose = () => {
        setAnchorEl(null);
    };

    const onClickReport = () => {
        sessionStorage.setItem(
            sessionKeyReportPersonMessageIdName,
            `${person_message_id}`
        );
        sessionStorage.setItem(sessionKeyReportSteamID, steamId);
        navigate('/report');
        handleClose();
    };

    return (
        <>
            <IconButton onClick={handleClick} size={'small'}>
                <SettingsSuggestIcon color={'info'} />
            </IconButton>
            <Menu
                id="chat-msg-menu"
                anchorEl={anchorEl}
                open={open}
                onClose={handleClose}
                anchorOrigin={{
                    vertical: 'top',
                    horizontal: 'left'
                }}
                transformOrigin={{
                    vertical: 'top',
                    horizontal: 'left'
                }}
            >
                <MenuItem onClick={onClickReport} disabled={flagged}>
                    <ListItemIcon>
                        <ReportIcon fontSize="small" color={'error'} />
                    </ListItemIcon>
                    <ListItemText>Create Report (Full)</ListItemText>
                </MenuItem>
                <MenuItem onClick={onClickReport} disabled={true}>
                    <ListItemIcon>
                        <ReportGmailerrorredIcon
                            fontSize="small"
                            color={'error'}
                        />
                    </ListItemIcon>
                    <ListItemText>Create Report (1-Click)</ListItemText>
                </MenuItem>
                <Divider />
                <MenuItem onClick={handleClose} disabled={true}>
                    <ListItemIcon>
                        <HistoryIcon fontSize="small" />
                    </ListItemIcon>
                    <ListItemText>Message Context</ListItemText>
                </MenuItem>
            </Menu>
        </>
    );
};
