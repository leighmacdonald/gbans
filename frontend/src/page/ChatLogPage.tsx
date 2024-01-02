import React, { useCallback, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import useUrlState from '@ahooksjs/use-url-state';
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
import { formatISO9075 } from 'date-fns/fp';
import { Formik } from 'formik';
import { FormikHelpers } from 'formik/dist/types';
import {
    apiGetMessages,
    defaultAvatarHash,
    PermissionLevel,
    PersonMessage,
    ServerSimple,
    sessionKeyReportPersonMessageIdName,
    sessionKeyReportSteamID
} from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { AutoRefreshField } from '../component/formik/AutoRefreshField';
import { AutoSubmitPaginationField } from '../component/formik/AutoSubmitPaginationField';
import { MessageField } from '../component/formik/MessageField';
import { PersonCellField } from '../component/formik/PersonCellField';
import { PersonanameField } from '../component/formik/PersonanameField';
import { ServerIDCell, ServerIDField } from '../component/formik/ServerIDField';
import { SteamIdField } from '../component/formik/SteamIdField';
import { ResetButton, SubmitButton } from '../component/modal/Buttons';
import { LazyTable, RowsPerPage } from '../component/table/LazyTable';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { useServers } from '../hooks/useServers';
import { logErr } from '../util/errors';

const anyServerSimple: ServerSimple = {
    server_name: 'Any',
    server_id: 0,
    server_name_long: 'Any',
    colour: ''
};

interface ChatLogFormValues {
    steam_id: string;
    personaname: string;
    message: string;
    auto_refresh: number;
    server_id: number;
}

export const ChatLogPage = () => {
    const [state, setState] = useUrlState({
        sortOrder: 'desc',
        sortColumn: 'person_message_id',
        page: '0',
        rowPerPageCount: `${RowsPerPage.TwentyFive}`,
        server: `${anyServerSimple.server_id}`,
        personaName: '',
        message: '',
        steamId: ''
    });

    const [rows, setRows] = useState<PersonMessage[]>([]);
    const [totalRows, setTotalRows] = useState<number>(0);
    const [loading, setLoading] = useState(false);
    const { currentUser } = useCurrentUserCtx();

    const { data: realServers } = useServers();

    const servers = useMemo(() => {
        return [anyServerSimple, ...realServers];
    }, [realServers]);

    const onSubmit = useCallback(
        (values: ChatLogFormValues) => {
            setLoading(true);
            apiGetMessages({
                server_id:
                    (values.server_id ?? anyServerSimple.server_id) > 0
                        ? values.server_id
                        : undefined,
                personaname: values.personaname,
                query: values.message,
                source_id: values.steam_id,
                limit: Number(state.rowPerPageCount),
                offset: Number(state.page ?? 0) * Number(state.rowPerPageCount),
                order_by: state.sortColumn,
                desc: state.sortOrder == 'desc'
            })
                .then((resp) => {
                    setRows(resp.data || []);
                    setTotalRows(resp.count);
                    setState({
                        server:
                            (values.server_id ?? 0) > 0
                                ? values.server_id
                                : undefined,
                        personaName: values.personaname,
                        message: values.message,
                        page:
                            Number(state.page) * Number(state.rowPerPageCount) >
                            resp.count
                                ? undefined
                                : state.page
                    });
                })
                .catch((e) => {
                    logErr(e);
                })
                .finally(() => {
                    setLoading(false);
                });
        },
        [
            setState,
            state.page,
            state.rowPerPageCount,
            state.sortColumn,
            state.sortOrder
        ]
    );

    const onReset = useCallback(
        async (
            _: ChatLogFormValues,
            formikHelpers: FormikHelpers<ChatLogFormValues>
        ) => {
            setState({
                sortOrder: 'desc',
                sortColumn: 'person_message_id',
                page: '0',
                rowPerPageCount: String(RowsPerPage.TwentyFive),
                server: String(anyServerSimple.server_id),
                personaName: '',
                message: '',
                steamId: ''
            });
            await formikHelpers.submitForm();
        },
        [setState]
    );

    return (
        <Formik<ChatLogFormValues>
            onSubmit={onSubmit}
            onReset={onReset}
            initialValues={{
                personaname: state.personaName,
                message: state.message,
                steam_id: state.steamId,
                auto_refresh: 0,
                server_id: state.server
            }}
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
                                    page={Number(state.page)}
                                    rowsPerPage={Number(state.rowPerPageCount)}
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
                            showPager
                            page={Number(state.page ?? '0')}
                            count={totalRows}
                            rowsPerPage={Number(state.rowPerPageCount)}
                            sortOrder={state.sortOrder}
                            sortColumn={state.sortColumn}
                            onSortColumnChanged={async (column) => {
                                setState({ sortColumn: column });
                            }}
                            onSortOrderChanged={async (direction) => {
                                setState({ sortOrder: direction });
                            }}
                            onRowsPerPageChange={(
                                event: React.ChangeEvent<
                                    HTMLInputElement | HTMLTextAreaElement
                                >
                            ) => {
                                setState({
                                    rowPerPageCount: Number(event.target.value),
                                    page: '0'
                                });
                            }}
                            onPageChange={(_, newPage) => {
                                setState({ page: `${newPage}` });
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
                                    hideSm: true,
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
                                                            row.auto_filter_flagged >
                                                            0
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
