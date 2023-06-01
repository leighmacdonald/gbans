import React, { useCallback, useEffect, useState } from 'react';
import Grid from '@mui/material/Grid';
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
import { DataTable } from '../component/DataTable';
import { steamIdQueryValue } from '../util/text';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import { Heading } from '../component/Heading';

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
    const [messages, setMessages] = useState<PersonMessage[]>([]);
    const [startDate, setStartDate] = useState<Date | null>(null);
    const [endDate, setEndDate] = useState<Date | null>(null);
    const [steamId, setSteamId] = useState<string>('');
    const [nameQuery, setNameQuery] = useState<string>('');
    const [messageQuery, setMessageQuery] = useState<string>('');
    const [servers, setServers] = useState<Server[]>([]);
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
        apiGetMessages({}).then((response) => {
            setMessages(response.result || []);
        });
    }, []);

    const onApply = useCallback(() => {
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
        apiGetMessages(opts).then((response) => {
            setMessages(response.result || []);
        });
    }, [endDate, messageQuery, nameQuery, selectedServer, startDate, steamId]);

    return (
        <Grid container spacing={2} paddingTop={3}>
            <Grid item xs={12}>
                <Paper elevation={1}>
                    <Stack>
                        <Heading>Chat History</Heading>

                        <Stack direction={'row'} spacing={2} padding={2}>
                            <TextField
                                placeholder={'Name'}
                                onChange={(evt) => {
                                    setNameQuery(evt.target.value);
                                }}
                            ></TextField>
                            <TextField
                                placeholder={'Steam ID'}
                                onChange={(evt) => {
                                    setSteamId(evt.target.value);
                                }}
                            ></TextField>
                            <TextField
                                placeholder={'Message'}
                                onChange={(evt) => {
                                    setMessageQuery(evt.target.value);
                                }}
                            ></TextField>
                            <Select<number>
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
                        </Stack>
                        <Stack direction={'row'} spacing={2} padding={2}>
                            <DesktopDatePicker
                                label="Date Start"
                                format={'MM/dd/yyyy'}
                                value={startDate}
                                onChange={(newValue: Date | null) => {
                                    setStartDate(newValue);
                                }}
                            />
                            <DesktopDatePicker
                                label="Date End"
                                format="MM/dd/yyyy"
                                value={endDate}
                                onChange={(newValue: Date | null) => {
                                    setEndDate(newValue);
                                }}
                            />
                            <ButtonGroup>
                                <Button
                                    variant={'contained'}
                                    color={'success'}
                                    onClick={onApply}
                                >
                                    Apply
                                </Button>
                                <Button variant={'contained'} color={'info'}>
                                    Reset
                                </Button>
                            </ButtonGroup>
                        </Stack>

                        <DataTable
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
                                            {row.steam_id.getSteamID64()}
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
                            rows={messages}
                        />
                    </Stack>
                </Paper>
            </Grid>
        </Grid>
    );
};
