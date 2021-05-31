import React, { useEffect, useMemo, useRef, useState } from 'react';
import SteamID from 'steamid';
import { formatRelative } from 'date-fns';
import {
    FormControl,
    Grid,
    Input,
    InputLabel,
    makeStyles,
    MenuItem,
    Select,
    TextField
} from '@material-ui/core';
import { log } from '../util/errors';
import { apiGetServers, Server } from '../util/api';
import {
    MsgType,
    SayEvt,
    ServerLog,
    StringIsNumber
} from '../util/game_events';
import useWebSocket, { ReadyState } from 'react-use-websocket';
import { takeRight } from 'lodash-es';
import { parseDateTime } from '../util/text';

const useStyles = makeStyles((theme) => ({
    formControl: {
        margin: theme.spacing(1),
        minWidth: 120,
        maxWidth: 300
    },
    chips: {
        display: 'flex',
        flexWrap: 'wrap'
    },
    chip: {
        margin: 2
    },
    noLabel: {
        marginTop: theme.spacing(3)
    }
}));

export const ServerLogView = (): JSX.Element => {
    const classes = useStyles();
    const proto = location.protocol === 'https:' ? 'wss' : 'ws';
    const port = location.port ? ':' + location.port : '';
    const messageHistory = useRef<ServerLog[]>([]);
    const [filterSteamID, setFilterSteamID] = useState<SteamID>(
        new SteamID('')
    );
    const [filteredMessages, setFilteredMessages] = useState<ServerLog[]>([]);
    const { sendJsonMessage, lastJsonMessage, readyState } = useWebSocket(
        `${proto}://${location.host}${port}/ws`,
        {
            onOpen: () => {
                sendJsonMessage({ token: localStorage.getItem('token') });
            },
            //Will attempt to reconnect on all close events, such as server shutting down
            shouldReconnect: () => true
        }
    );

    messageHistory.current = useMemo(() => {
        messageHistory.current.push(lastJsonMessage);
        return messageHistory.current;
    }, [lastJsonMessage]);

    const connectionStatus = {
        [ReadyState.CONNECTING]: 'Connecting',
        [ReadyState.OPEN]: 'Open',
        [ReadyState.CLOSING]: 'Closing',
        [ReadyState.CLOSED]: 'Closed',
        [ReadyState.UNINSTANTIATED]: 'Uninstantiated'
    }[readyState];

    const [servers, setServers] = useState<Server[]>([]);
    const [renderLimit, setRenderLimit] = useState<number>(10000);
    const [filterServerIDs, setFilterServerIDs] = useState<number[]>([]);
    const [filterMsgTypes, setFilterMsgTypes] = useState<MsgType[]>([
        MsgType.Any
    ]);
    useEffect(() => {
        async function fn() {
            const servers = await apiGetServers();
            if (
                servers !== null &&
                Object.prototype.hasOwnProperty.call(servers, 'error')
            ) {
                log(`Error fetching servers`);
                setServers([]);
                return;
            }
            setServers(servers as Server[]);
        }

        // noinspection JSIgnoredPromiseFromCall
        fn();
    }, []);

    const handleChangeFilterMsg = (event: any) => {
        const v = event.target.value.filter(StringIsNumber);
        setFilterMsgTypes(v);
    };

    const handleChangeServers = (event: any) => {
        setFilterServerIDs(event.target.value);
    };
    const handleChangeRenderLimit = (event: any) => {
        setRenderLimit(event.target.value);
    };

    const onFilterSteamIDChange = (event: any) => {
        setFilterSteamID(new SteamID(event.target.value));
    };

    useEffect(() => {
        let logs = messageHistory.current.filter((v) => v);
        if (filterServerIDs.length > 0) {
            logs = logs.filter((s) => filterServerIDs.includes(s.server_id));
        }
        if (filterSteamID.isValid()) {
            logs = logs.filter(
                (s) => s.source_id == filterSteamID.getSteamID64()
            );
        }
        if (
            filterMsgTypes.length > 0 &&
            !filterMsgTypes.includes(MsgType.Any)
        ) {
            logs = logs.filter((s) => filterMsgTypes.includes(s.event_type));
        }
        logs = takeRight<ServerLog>(logs, renderLimit);
        setFilteredMessages(logs);
    }, [
        setFilterServerIDs,
        setFilterMsgTypes,
        setRenderLimit,
        lastJsonMessage
    ]);

    return (
        <Grid container>
            <Grid item xs={3}>
                <FormControl className={classes.formControl}>
                    <TextField
                        onChange={onFilterSteamIDChange}
                        id="standard-basic"
                        label="SteamID"
                    />
                </FormControl>
            </Grid>
            <Grid item xs={3}>
                <FormControl className={classes.formControl}>
                    <InputLabel id="limit-filters-label">
                        Limit results
                    </InputLabel>
                    <Select
                        labelId="limit-filters-label"
                        id="limit-filters"
                        value={renderLimit}
                        defaultValue={25}
                        onChange={handleChangeRenderLimit}
                    >
                        <MenuItem value={25}>25</MenuItem>
                        <MenuItem value={100}>100</MenuItem>
                        <MenuItem value={1000}>1000</MenuItem>
                        <MenuItem value={5000}>5000</MenuItem>
                        <MenuItem value={10000}>10000</MenuItem>
                        <MenuItem value={Number.MAX_SAFE_INTEGER}>
                            inf.
                        </MenuItem>
                    </Select>
                </FormControl>
            </Grid>
            <Grid item xs={3}>
                <FormControl className={classes.formControl}>
                    <InputLabel id="msg-filters-label">
                        Message Filters
                    </InputLabel>
                    <Select
                        labelId="msg-filters-label"
                        id="demo-mutiple-name"
                        multiple
                        value={filterMsgTypes}
                        onChange={handleChangeFilterMsg}
                        input={<Input />}
                    >
                        {Object.values(MsgType)
                            .filter(StringIsNumber)
                            .map((mt) => (
                                <MenuItem key={mt} value={mt}>
                                    {MsgType[mt as number]}
                                </MenuItem>
                            ))}
                    </Select>
                </FormControl>
            </Grid>
            <Grid item xs={3}>
                <FormControl className={classes.formControl}>
                    <InputLabel id="server-filters-label">
                        Server Filters
                    </InputLabel>
                    <Select
                        labelId="server-filters-label"
                        id="server-filters"
                        multiple
                        value={filterServerIDs}
                        onChange={handleChangeServers}
                        input={<Input />}
                    >
                        {servers.map((s) => (
                            <MenuItem key={s.server_id} value={s.server_id}>
                                {s.server_name}
                            </MenuItem>
                        ))}
                    </Select>
                </FormControl>
            </Grid>
            <Grid item xs={12}>
                <h5>Connection Status: {connectionStatus}</h5>
            </Grid>
            <Grid item xs={12}>
                <Grid container>
                    {filteredMessages.map((msg, i) => renderServerLog(msg, i))}
                </Grid>
            </Grid>
        </Grid>
    );
};

export const renderServerLog = (l: ServerLog, i: number): JSX.Element => {
    if (!l) {
        return <></>;
    }
    let v: JSX.Element;
    switch (l.event_type) {
        case MsgType.Say: {
            const e = l.payload as SayEvt;
            v = (
                <Grid container>
                    <Grid item xs={2}>
                        {formatRelative(
                            new Date(),
                            parseDateTime(e.created_on)
                        )}
                    </Grid>
                    <Grid item xs={2}>
                        {e.source ? e.source.sid : '???'}
                    </Grid>
                    <Grid item xs={8}>
                        {e.msg}
                    </Grid>
                </Grid>
            );
            break;
        }
        default:
            v = <div>{JSON.stringify(l.payload)}</div>;
    }
    return (
        <Grid key={`sl-${i}`} item xs={12}>
            {v}
        </Grid>
    );
};
