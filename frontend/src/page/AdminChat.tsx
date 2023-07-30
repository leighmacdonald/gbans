import React, { useEffect, useState } from 'react';
import Grid from '@mui/material/Unstable_Grid2';
import Paper from '@mui/material/Paper';
import Typography from '@mui/material/Typography';
import Select from '@mui/material/Select';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import { DesktopDatePicker } from '@mui/x-date-pickers/DesktopDatePicker';
import MenuItem from '@mui/material/MenuItem';
import {
    apiGetMessages,
    apiGetServers,
    MessageQuery,
    PersonMessage,
    Server
} from '../api';
import { steamIdQueryValue } from '../util/text';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import { Heading } from '../component/Heading';
import { LazyTable } from '../component/LazyTable';
import { logErr } from '../util/errors';
import { Order, RowsPerPage } from '../component/DataTable';
import { formatISO9075 } from 'date-fns/fp';
import { TablePagination } from '@mui/material';
import { useTimer } from 'react-timer-hook';
import ChatIcon from '@mui/icons-material/Chat';
import FilterAltIcon from '@mui/icons-material/FilterAlt';

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
    updated_on: new Date(),
    created_on: new Date()
};

export interface DelayedTextInputProps {
    delay?: number;
    onChange: (value: string) => void;
    placeholder: string;
}

export const DelayedTextInput = ({
    delay,
    onChange,
    placeholder
}: DelayedTextInputProps) => {
    const [value, setValue] = useState<string>('');
    const { restart } = useTimer({
        autoStart: false,
        expiryTimestamp: new Date(),
        onExpire: () => {
            onChange(value.length <= 2 ? '' : value);
            console.log(value);
        }
    });

    const onInputChange = (
        event: React.ChangeEvent<HTMLTextAreaElement | HTMLInputElement>
    ) => {
        setValue(event.target.value);
        const time = new Date();
        time.setSeconds(time.getSeconds() + (delay ?? 2));
        restart(time, true);
    };

    return (
        <TextField
            fullWidth
            value={value}
            placeholder={placeholder}
            onChange={onInputChange}
        />
    );
};

export const AdminChat = () => {
    const [startDate, setStartDate] = useState<Date | null>(null);
    const [endDate, setEndDate] = useState<Date | null>(null);
    const [steamId, setSteamId] = useState<string>('');
    const [nameQuery, setNameQuery] = useState<string>('');
    const [messageQuery, setMessageQuery] = useState<string>('');
    const [servers, setServers] = useState<Server[]>([]);
    const [rows, setRows] = useState<PersonMessage[]>([]);
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] =
        useState<keyof PersonMessage>('person_message_id');
    const [page, setPage] = useState(0);
    const [rowPerPageCount, setRowPerPageCount] = useState<number>(
        RowsPerPage.Fifty
    );
    const [totalRows, setTotalRows] = useState<number>(0);
    //const [pageCount, setPageCount] = useState<number>(0);

    const [selectedServer, setSelectedServer] = useState<number>(
        anyServer.server_id
    );

    useEffect(() => {
        apiGetServers().then((resp) => {
            if (!resp.status || !resp.result) {
                return;
            }
            setServers([
                anyServer,
                ...resp.result.sort((a, b) => {
                    return a.server_name.localeCompare(b.server_name);
                })
            ]);
        });
    }, []);

    useEffect(() => {
        const opts: MessageQuery = {};
        if (selectedServer > 0) {
            opts.server_id = selectedServer;
        }
        if (nameQuery) {
            opts.persona_name = nameQuery;
        }
        if (messageQuery) {
            opts.query = messageQuery;
        }
        if (steamId) {
            opts.steam_id = steamId;
        }
        if (startDate) {
            opts.sent_after = startDate;
        }
        if (endDate) {
            opts.sent_before = endDate;
        }
        opts.limit = rowPerPageCount;
        opts.offset = page * rowPerPageCount;
        opts.order_by = sortColumn;
        opts.desc = sortOrder == 'desc';
        apiGetMessages(opts)
            .then((resp) => {
                const count = resp.result?.total_messages || 0;
                setRows(resp.result?.messages || []);
                setTotalRows(count);
            })
            .catch((e) => {
                logErr(e);
            });
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

    return (
        <Grid container spacing={2} paddingTop={3}>
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
                                    placeholder={'Name'}
                                    onChange={(value) => {
                                        setNameQuery(value);
                                    }}
                                />
                            </Grid>
                            <Grid xs={6} md={3}>
                                <DelayedTextInput
                                    placeholder={'Steam ID'}
                                    onChange={(value) => {
                                        setSteamId(value);
                                    }}
                                />
                            </Grid>
                            <Grid xs={6} md={3}>
                                <DelayedTextInput
                                    placeholder={'Message'}
                                    onChange={(value) => {
                                        setMessageQuery(value);
                                    }}
                                />
                            </Grid>
                            <Grid xs={6} md={3}>
                                <Select<number>
                                    fullWidth
                                    value={selectedServer}
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
                                    value={startDate}
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
                                    value={endDate}
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
                                    >
                                        Reset
                                    </Button>
                                </ButtonGroup>
                            </Grid>
                        </Grid>
                    </Stack>
                </Paper>
            </Grid>

            <Grid xs={12}>
                <TablePagination
                    component="div"
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
                        setRowPerPageCount(parseInt(event.target.value, 10));
                        setPage(0);
                    }}
                    onPageChange={(_, newPage) => {
                        setPage(newPage);
                    }}
                />
            </Grid>
            <Grid xs={12}>
                <Heading iconLeft={<ChatIcon />}>Chat Messages</Heading>
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
                            queryValue: (o) =>
                                `${o.server_id} + ${o.server_name}`,
                            renderer: (row) => (
                                <Typography variant={'button'}>
                                    {row.server_name}
                                </Typography>
                            )
                        },
                        {
                            label: 'Created',
                            tooltip: 'Time the message was sent',
                            sortKey: 'created_on',
                            sortType: 'date',
                            align: 'center',
                            width: 180,
                            queryValue: (o) => steamIdQueryValue(o.steam_id),
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
                            queryValue: (o) => `${o.persona_name}`,
                            renderer: (row) => (
                                <Typography variant={'body2'}>
                                    {row.persona_name}
                                </Typography>
                            )
                        },
                        {
                            label: 'Message',
                            tooltip: 'Message',
                            sortKey: 'body',
                            align: 'left',
                            queryValue: (o) => o.body
                        }
                    ]}
                    rows={rows}
                />
            </Grid>
        </Grid>
    );
};
