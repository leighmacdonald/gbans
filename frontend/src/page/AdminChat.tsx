import React, { useCallback, useEffect, useState } from 'react';
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

export const AdminChat = () => {
    const [startDate, setStartDate] = useState<Date | null>(null);
    const [endDate, setEndDate] = useState<Date | null>(null);
    const [steamId, setSteamId] = useState<string>('');
    const [nameQuery, setNameQuery] = useState<string>('');
    const [messageQuery, setMessageQuery] = useState<string>('');
    const [servers, setServers] = useState<Server[]>([]);
    const [rows, setRows] = useState<PersonMessage[]>([]);
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const [_, setTotalRows] = useState<number>(0);
    const [selectedServer, setSelectedServer] = useState<number>(
        anyServer.server_id
    );

    useEffect(() => {
        apiGetServers().then((resp) => {
            if (!resp.status || !resp.result) {
                return;
            }
            setServers([anyServer, ...resp.result]);
        });
    }, []);

    const onApply = useCallback(async () => {
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
        try {
            const resp = await apiGetMessages(opts);
            setRows(resp.result?.messages || []);
            setTotalRows(resp.result?.totalMessages || 0);
        } catch (e) {
            logErr(e);
        }
    }, [endDate, messageQuery, nameQuery, selectedServer, startDate, steamId]);

    return (
        <Grid container spacing={2} paddingTop={3}>
            <Grid xs={12}>
                <Paper elevation={1}>
                    <Stack>
                        <Heading>Chat History</Heading>

                        <Grid
                            container
                            padding={2}
                            spacing={2}
                            justifyContent={'center'}
                            alignItems={'center'}
                        >
                            <Grid xs={6} md={3}>
                                <TextField
                                    fullWidth
                                    placeholder={'Name'}
                                    onChange={(evt) => {
                                        setNameQuery(evt.target.value);
                                    }}
                                />
                            </Grid>
                            <Grid xs={6} md={3}>
                                <TextField
                                    fullWidth
                                    placeholder={'Steam ID'}
                                    onChange={(evt) => {
                                        setSteamId(evt.target.value);
                                    }}
                                ></TextField>
                            </Grid>
                            <Grid xs={6} md={3}>
                                <TextField
                                    fullWidth
                                    placeholder={'Message'}
                                    onChange={(evt) => {
                                        setMessageQuery(evt.target.value);
                                    }}
                                ></TextField>
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
                                        color={'success'}
                                        onClick={onApply}
                                    >
                                        Apply
                                    </Button>
                                    <Button
                                        variant={'contained'}
                                        color={'info'}
                                    >
                                        Reset
                                    </Button>
                                </ButtonGroup>
                            </Grid>
                        </Grid>

                        <LazyTable<PersonMessage>
                            columns={[
                                {
                                    label: 'ID',
                                    tooltip: 'ID',
                                    sortKey: 'server_id',
                                    queryValue: (o) =>
                                        `${o.server_id} + ${o.server_name}`,
                                    renderer: (row) => (
                                        <Button
                                            variant={'text'}
                                            fullWidth
                                            onClick={() => {
                                                setSelectedServer(
                                                    row.server_id
                                                );
                                            }}
                                        >
                                            {row.server_name}
                                        </Button>
                                    )
                                },
                                {
                                    label: 'Steam ID',
                                    tooltip: 'Steam ID',
                                    sortKey: 'steam_id',
                                    queryValue: (o) =>
                                        steamIdQueryValue(o.steam_id),
                                    renderer: (row) => (
                                        <Typography variant={'body1'}>
                                            {row.steam_id}
                                        </Typography>
                                    )
                                },
                                {
                                    label: 'Name',
                                    tooltip: 'Persona Name',
                                    sortKey: 'persona_name',
                                    queryValue: (o) => `${o.persona_name}`
                                },
                                {
                                    label: 'Message',
                                    tooltip: 'Message',
                                    sortKey: 'body',
                                    queryValue: (o) => o.body
                                }
                            ]}
                            defaultSortColumn={'created_on'}
                            rowsPerPage={100}
                            rows={rows}
                        />
                    </Stack>
                </Paper>
            </Grid>
        </Grid>
    );
};
