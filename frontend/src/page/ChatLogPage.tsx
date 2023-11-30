import React, { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import ChatIcon from '@mui/icons-material/Chat';
import FilterAltIcon from '@mui/icons-material/FilterAlt';
import FlagIcon from '@mui/icons-material/Flag';
import HistoryIcon from '@mui/icons-material/History';
import ReportIcon from '@mui/icons-material/Report';
import ReportGmailerrorredIcon from '@mui/icons-material/ReportGmailerrorred';
import SettingsSuggestIcon from '@mui/icons-material/SettingsSuggest';
import { Divider, IconButton } from '@mui/material';
import ButtonGroup from '@mui/material/ButtonGroup';
import CircularProgress from '@mui/material/CircularProgress';
import ListItemIcon from '@mui/material/ListItemIcon';
import ListItemText from '@mui/material/ListItemText';
import Menu from '@mui/material/Menu';
import MenuItem from '@mui/material/MenuItem';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { parseISO } from 'date-fns';
import { formatISO9075 } from 'date-fns/fp';
import { Formik } from 'formik';
import {
    apiGetMessages,
    apiGetServers,
    defaultAvatarHash,
    PermissionLevel,
    PersonMessage,
    Server,
    ServerSimple,
    sessionKeyReportPersonMessageIdName,
    sessionKeyReportSteamID
} from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { AutoRefreshField } from '../component/formik/AutoRefreshField';
import { AutoSubmitPaginationField } from '../component/formik/AutoSubmitPaginationField';
import { DateEndField } from '../component/formik/DateEndField';
import { DateStartField } from '../component/formik/DateStartField';
import { MessageField } from '../component/formik/MessageField';
import { PersonCellField } from '../component/formik/PersonCellField';
import { PersonanameField } from '../component/formik/PersonanameField';
import { ServerIDCell, ServerIDField } from '../component/formik/ServerIDField';
import { SteamIdField } from '../component/formik/SteamIdField';
import { ResetButton, SubmitButton } from '../component/modal/Buttons';
import { LazyTable, Order, RowsPerPage } from '../component/table/LazyTable';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { logErr } from '../util/errors';
import { Nullable } from '../util/types';

const anyServer: Server = {
    short_name: 'Any',
    server_id: 0,
    name: 'Any',
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
    enable_stats: true,
    log_secret: 0,
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
    startDate: Nullable<Date>;
    endDate: Nullable<Date>;
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
        rowPerPageCount: RowsPerPage.TwentyFive,
        nameQuery: '',
        messageQuery: '',
        steamId: '',
        page: 0
    };
    const item = localStorage.getItem(localStorageKey);
    if (item) {
        config = JSON.parse(item);
        if (config.startDate) {
            config.startDate = parseISO(config.startDate as unknown as string);
        }
        if (config.endDate) {
            config.endDate = parseISO(config.endDate as unknown as string);
        }
    }
    return config;
};

interface ChatLogFormValues {
    date_start: Nullable<Date>;
    date_end: Nullable<Date>;
    steam_id: string;
    personaname: string;
    message: string;
    auto_refresh: number;
    server_id: number;
}

export const ChatLogPage = () => {
    const init = loadState();
    const [sortOrder, setSortOrder] = useState<Order>(init.sortOrder);
    const [sortColumn, setSortColumn] = useState<keyof PersonMessage>(
        init.sortColumn
    );
    const [rows, setRows] = useState<PersonMessage[]>([]);
    const [page, setPage] = useState(init.page);
    const [rowPerPageCount, setRowPerPageCount] = useState<number>(
        init.rowPerPageCount
    );
    const [totalRows, setTotalRows] = useState<number>(0);
    const [loading, setLoading] = useState(false);
    const [servers, setServers] = useState<ServerSimple[]>([]);
    const { currentUser } = useCurrentUserCtx();

    const saveState = useCallback(
        (values: ChatLogFormValues) => {
            localStorage.setItem(
                localStorageKey,
                JSON.stringify({
                    endDate: values.date_end,
                    steamId: values.steam_id,
                    messageQuery: values.message,
                    nameQuery: values.personaname,
                    page,
                    rowPerPageCount,
                    selectedServer: values.server_id,
                    sortColumn,
                    sortOrder,
                    startDate: values.date_start
                } as ChatQueryState<PersonMessage>)
            );
        },
        [page, rowPerPageCount, sortColumn, sortOrder]
    );

    useEffect(() => {
        const abortController = new AbortController();

        apiGetServers(abortController)
            .then((resp) => {
                setServers([
                    anyServerSimple,
                    ...resp.sort((a: ServerSimple, b: ServerSimple) => {
                        return a.server_name.localeCompare(b.server_name);
                    })
                ]);
            })
            .then(() => {
                onSubmit(iv);
            })
            .catch(logErr);

        return () => abortController.abort();
        // eslint-disable-next-line react-hooks/exhaustive-deps
    }, []);

    const onSubmit = useCallback(
        (values: ChatLogFormValues) => {
            setLoading(true);
            apiGetMessages({
                server_id: values.server_id > 0 ? values.server_id : undefined,
                personaname: values.personaname,
                query: values.message,
                source_id: values.steam_id,
                date_start: values.date_start ?? undefined,
                date_end: values.date_end ?? undefined,
                limit: rowPerPageCount,
                offset: page * rowPerPageCount,
                order_by: sortColumn,
                desc: sortOrder == 'desc'
            })
                .then((resp) => {
                    setRows(resp.data || []);
                    setTotalRows(resp.count);
                    if (page * rowPerPageCount > resp.count) {
                        setPage(0);
                    }
                })
                .catch((e) => {
                    logErr(e);
                })
                .finally(() => {
                    setLoading(false);
                });

            saveState(values);
        },
        [page, rowPerPageCount, sortColumn, sortOrder, saveState]
    );
    const iv: ChatLogFormValues = {
        personaname: '',
        message: '',
        steam_id: '',
        date_start: null,
        date_end: null,
        auto_refresh: 0,
        server_id: 0
    };

    const onReset = () => {
        setPage(0);
        setSortColumn('person_message_id');
        setSortOrder('desc');
    };

    return (
        <Formik<ChatLogFormValues>
            onSubmit={onSubmit}
            initialValues={iv}
            onReset={onReset}
        >
            <Grid container spacing={2}>
                <Grid xs={12}>
                    <ContainerWithHeader
                        title={'Chat Filters'}
                        iconLeft={<FilterAltIcon />}
                    >
                        <Grid
                            container
                            padding={2}
                            spacing={2}
                            justifyContent={'center'}
                            alignItems={'center'}
                        >
                            <Grid xs={6} md={3}>
                                <AutoSubmitPaginationField
                                    page={page}
                                    rowsPerPage={rowPerPageCount}
                                />
                                <PersonanameField />
                            </Grid>
                            <Grid xs={6} md={3}>
                                <SteamIdField />
                            </Grid>
                            <Grid xs={6} md={3}>
                                <MessageField />
                            </Grid>
                            <Grid xs={6} md={3}>
                                <ServerIDField servers={servers} />
                            </Grid>
                            <Grid xs={6} md={3}>
                                <DateStartField />
                            </Grid>
                            <Grid xs={6} md={3}>
                                <DateEndField />
                            </Grid>
                            <Grid xs={3}>
                                {currentUser.permission_level >=
                                    PermissionLevel.Moderator && (
                                    <AutoRefreshField />
                                )}
                            </Grid>
                            <Grid xs md={3} mdOffset="auto">
                                <ButtonGroup size={'large'} fullWidth>
                                    <ResetButton />
                                    <SubmitButton label={'Apply'} />
                                </ButtonGroup>
                            </Grid>
                        </Grid>
                    </ContainerWithHeader>
                </Grid>
                <Grid xs={12}>
                    <Grid
                        xs={12}
                        container
                        justifyContent="space-between"
                        alignItems="center"
                        flexDirection={{ xs: 'column', sm: 'row' }}
                    >
                        <Grid xs={3}></Grid>
                    </Grid>

                    <Grid xs={12}>
                        <ContainerWithHeader
                            iconLeft={
                                loading ? (
                                    <CircularProgress
                                        color="inherit"
                                        size={20}
                                    />
                                ) : (
                                    <ChatIcon />
                                )
                            }
                            title={'Chat Logs'}
                        >
                            <LazyTable<PersonMessage>
                                showPager
                                page={page}
                                count={totalRows}
                                rowsPerPage={rowPerPageCount}
                                sortOrder={sortOrder}
                                sortColumn={sortColumn}
                                onSortColumnChanged={async (column) => {
                                    setSortColumn(column);
                                }}
                                onSortOrderChanged={async (direction) => {
                                    setSortOrder(direction);
                                }}
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
                                columns={[
                                    {
                                        label: 'Server',
                                        tooltip: 'Server',
                                        sortKey: 'server_id',
                                        align: 'center',
                                        width: 120,
                                        renderer: (row) => (
                                            <ServerIDCell
                                                server_id={row.server_id}
                                                server_name={row.server_name}
                                            />
                                        )
                                    },
                                    {
                                        label: 'Created',
                                        tooltip: 'Time the message was sent',
                                        sortKey: 'created_on',
                                        sortType: 'date',
                                        sortable: false,
                                        align: 'center',
                                        width: 220,
                                        renderer: (row) => (
                                            <Typography variant={'body1'}>
                                                {`${formatISO9075(
                                                    row.created_on
                                                )}`}
                                            </Typography>
                                        )
                                    },
                                    {
                                        label: 'Name',
                                        tooltip: 'Persona Name',
                                        sortKey: 'persona_name',
                                        width: 250,
                                        align: 'left',
                                        renderer: (row) => (
                                            <PersonCellField
                                                steam_id={row.steam_id}
                                                avatar_hash={
                                                    row.avatar_hash != ''
                                                        ? row.avatar_hash
                                                        : defaultAvatarHash
                                                }
                                                personaname={row.persona_name}
                                            />
                                        )
                                    },
                                    {
                                        label: 'Message',
                                        tooltip: 'Message',
                                        sortKey: 'body',
                                        align: 'left',
                                        renderer: (row) => {
                                            return (
                                                <Grid container>
                                                    <Grid xs padding={1}>
                                                        <Typography
                                                            variant={'body1'}
                                                        >
                                                            {row.body}
                                                        </Typography>
                                                    </Grid>

                                                    {row.auto_filter_flagged >
                                                        0 && (
                                                        <Grid
                                                            xs={'auto'}
                                                            padding={1}
                                                            display="flex"
                                                            justifyContent="center"
                                                            alignItems="center"
                                                        >
                                                            <>
                                                                <FlagIcon
                                                                    color={
                                                                        'error'
                                                                    }
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
                                                                row.auto_filter_flagged >
                                                                0
                                                            }
                                                            steamId={
                                                                row.steam_id
                                                            }
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
            </Grid>
        </Formik>
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
