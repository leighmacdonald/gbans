import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { Heading } from '../component/Heading';
import Paper from '@mui/material/Paper';
import Grid from '@mui/material/Grid';
import { DataTable, RowsPerPage } from '../component/DataTable';
import {
    apiServerQuery,
    qpBaseQuery,
    qpLobby,
    qpMsgJoinLobby,
    qpMsgType,
    qpUserMessage,
    SlimServer,
    UserProfile
} from '../api';
import Button from '@mui/material/Button';
import Link from '@mui/material/Link';
import Stack from '@mui/material/Stack';
import Switch from '@mui/material/Switch';
import TextField from '@mui/material/TextField';
import FormGroup from '@mui/material/FormGroup';
import FormControlLabel from '@mui/material/FormControlLabel';
import useWebSocket, { ReadyState } from 'react-use-websocket';
import Typography from '@mui/material/Typography';

export const QuickPlayPage = (): JSX.Element => {
    const [allServers, setAllServers] = useState<SlimServer[]>([]);
    const [minPlayers, setMinPlayers] = useState<number>(0);
    const [maxPlayers, setMaxPlayers] = useState<number>(0);
    const [notFull, setNotFull] = useState<boolean>(true);
    const [chatInput, setChatInput] = useState<string>('');
    const [lobby, setLobby] = useState<qpLobby>({ lobby_id: '', clients: [] });
    const [messageHistory, setMessageHistory] = useState<qpUserMessage[]>([]);

    const token = useMemo(() => {
        return localStorage.getItem('token') ?? '';
    }, []);

    const socketUrl = useMemo(() => {
        const parsedUrl = new URL(window.location.href);
        return `${parsedUrl.protocol == 'https' ? 'wss' : 'ws'}://${
            parsedUrl.host
        }/ws/quickplay`;
    }, []);

    const { readyState, lastJsonMessage, sendJsonMessage } = useWebSocket(
        socketUrl,
        {
            onError: (event: WebSocketEventMap['error']) => {
                setMessageHistory((prevState) =>
                    prevState.concat({
                        message: event.type,
                        created_at: new Date().toISOString()
                    })
                );
            },
            onClose: (_: WebSocketEventMap['close']) => {
                setMessageHistory((prevState) =>
                    prevState.concat({
                        message: 'Lobby connection closed',
                        created_at: new Date().toISOString()
                    })
                );
            },
            queryParams: {
                token: token
            },
            onOpen: (_: WebSocketEventMap['open']) => {
                setMessageHistory((prevState) =>
                    prevState.concat({
                        message: 'Lobby connection opened',
                        created_at: new Date().toISOString()
                    })
                );
            }, //Will attempt to reconnect on all close events, such as server shutting down
            shouldReconnect: () => true
        }
    );

    const sendMessage = useCallback(() => {
        if (chatInput == '') {
            return;
        }
        const req: qpBaseQuery = {
            msg_type: qpMsgType.qpMsgTypeSendMsg,
            payload: {
                message: chatInput,
                created_at: new Date().toISOString()
            } as qpUserMessage
        };
        sendJsonMessage(req);
        setChatInput('');
    }, [chatInput, sendJsonMessage]);

    useEffect(() => {
        if (lastJsonMessage != null) {
            //setMessageHistory((prev) => prev.concat(lastJsonMessage));
            const p = lastJsonMessage as unknown as qpBaseQuery;
            switch (p.msg_type) {
                case qpMsgType.qpMsgTypeJoinLobby: {
                    const payload = p.payload as qpMsgJoinLobby;
                    setLobby(payload.lobby);
                    return;
                }
                case qpMsgType.qpMsgTypeSendMsg: {
                    const msgPayload = p.payload as qpUserMessage;
                    setMessageHistory((prev) => prev.concat(msgPayload));
                    return;
                }
                default: {
                    console.log(lastJsonMessage);
                }
            }
        }
    }, [lastJsonMessage]);

    const connectionStatus = {
        [ReadyState.CONNECTING]: 'Connecting',
        [ReadyState.OPEN]: 'Open',
        [ReadyState.CLOSING]: 'Closing',
        [ReadyState.CLOSED]: 'Closed',
        [ReadyState.UNINSTANTIATED]: 'Uninstantiated'
    }[readyState];

    useEffect(() => {
        apiServerQuery({
            gameTypes: []
        }).then((resp) => {
            if (!resp.status) {
                return;
            }
            setAllServers(resp.result ?? []);
        });
    }, []);

    const msgs = useMemo(() => {
        const m = messageHistory;
        m.reverse();
        return m;
    }, [messageHistory]);

    const filteredServers = useMemo(() => {
        let servers = allServers;
        if (notFull) {
            servers = servers.filter((s) => s.players < s.max_players);
        }
        if (minPlayers > 0) {
            servers = servers.filter((s) => s.players >= minPlayers);
        }
        if (maxPlayers > 0) {
            servers = servers.filter((s) => s.players <= maxPlayers);
        }
        return servers;
    }, [maxPlayers, minPlayers, allServers, notFull]);

    const dataTable = useMemo(() => {
        return (
            <DataTable
                columns={[
                    {
                        label: 'Name',
                        sortKey: 'name',
                        tooltip: 'Server Name',
                        sortable: true,
                        align: 'left',
                        queryValue: (row) => row.name
                    },
                    {
                        label: 'Address',
                        sortKey: 'addr',
                        tooltip: 'Address',
                        align: 'left',
                        sortable: true
                    },
                    {
                        label: 'Map',
                        sortKey: 'map',
                        tooltip: 'Map',
                        sortable: true,
                        queryValue: (row) => row.map
                    },
                    {
                        label: 'Players',
                        sortKey: 'players',
                        tooltip: 'Players',
                        sortable: true,
                        queryValue: (row) => `${row.players}`,
                        renderer: (row) => {
                            return `${row.players}/${row.max_players}`;
                        }
                    },
                    {
                        label: 'Actions',
                        tooltip: 'Actions',
                        virtual: true,
                        virtualKey: 'act',
                        renderer: (row) => {
                            return (
                                <Button
                                    variant={'contained'}
                                    component={Link}
                                    href={`steam://connect/${row.addr}`}
                                >
                                    Connect
                                </Button>
                            );
                        }
                    }
                ]}
                defaultSortColumn={'name'}
                rowsPerPage={RowsPerPage.TwentyFive}
                rows={filteredServers}
            />
        );
    }, [filteredServers]);

    return (
        <Grid container paddingTop={3} spacing={2}>
            <Grid item xs={12}>
                <Grid container spacing={2}>
                    <Grid item xs={8}>
                        <Paper elevation={1}>
                            <Stack spacing={1}>
                                <Heading>{`Lobby (status: ${connectionStatus})`}</Heading>
                                <Stack
                                    direction={'column-reverse'}
                                    sx={{
                                        height: 200,
                                        overflow: 'scroll'
                                    }}
                                >
                                    {msgs.map((msg, i) => {
                                        return (
                                            <Typography
                                                key={`msg-${i}`}
                                                variant={'body2'}
                                            >
                                                {msg.created_at} --{' '}
                                                {msg.steam_id ?? '__lobby__'} --
                                                {msg.message}
                                            </Typography>
                                        );
                                    })}
                                </Stack>
                                <Stack direction={'row'}>
                                    <TextField
                                        fullWidth
                                        value={chatInput}
                                        onChange={(evt) => {
                                            setChatInput(evt.target.value);
                                        }}
                                    />
                                    <Button
                                        color={'success'}
                                        variant={'contained'}
                                        onClick={sendMessage}
                                    >
                                        Send
                                    </Button>
                                </Stack>
                            </Stack>
                        </Paper>
                    </Grid>
                    <Grid item xs={4}>
                        <Paper elevation={1}>
                            <Heading>{`Lobby Members (lobby: ${lobby.lobby_id})`}</Heading>
                            <Stack
                                sx={{
                                    height: 200,
                                    overflow: 'scroll'
                                }}
                            >
                                {lobby.clients.map((client) => {
                                    const user =
                                        client.user as unknown as UserProfile;
                                    return (
                                        <Typography
                                            align={'center'}
                                            padding={1}
                                            variant={'h5'}
                                            key={`client-${user.steam_id.toString()}`}
                                        >
                                            {user.steam_id.toString()}
                                        </Typography>
                                    );
                                })}
                            </Stack>
                        </Paper>
                    </Grid>
                </Grid>
            </Grid>
            <Grid item xs={12}>
                <Paper elevation={1}>
                    <Heading>Quickplay Filters</Heading>
                    <Stack spacing={1} direction={'row'} padding={2}>
                        <TextField
                            id="outlined-basic"
                            label="Min Players"
                            variant="outlined"
                            type={'number'}
                            value={minPlayers}
                            onChange={(evt) => {
                                const value = parseInt(evt.target.value);
                                if (value && value > 31) {
                                    return;
                                }
                                if (maxPlayers > 0 && value > maxPlayers) {
                                    setMaxPlayers(value);
                                }
                                setMinPlayers(value ?? 0);
                            }}
                        />
                        <TextField
                            id="outlined-basic"
                            label="Max Players"
                            variant="outlined"
                            type={'number'}
                            value={maxPlayers}
                            onChange={(evt) => {
                                let value = parseInt(evt.target.value);
                                if (value && value > 32) {
                                    return;
                                }
                                if (value < minPlayers) {
                                    value = minPlayers;
                                }
                                setMaxPlayers(value ?? 0);
                            }}
                        />
                        <FormGroup>
                            <FormControlLabel
                                control={
                                    <Switch
                                        defaultChecked
                                        value={notFull}
                                        onChange={(_, checked) => {
                                            setNotFull(checked);
                                        }}
                                    />
                                }
                                label="Hide Full"
                            />
                        </FormGroup>
                    </Stack>
                </Paper>
            </Grid>
            <Grid item xs={12}>
                <Paper elevation={1}>
                    <Heading>Community Servers</Heading>
                    {dataTable}
                </Paper>
            </Grid>
        </Grid>
    );
};
