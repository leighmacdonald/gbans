import React, {
    SyntheticEvent,
    useEffect,
    useMemo,
    useRef,
    useState
} from 'react';
import { Grid } from '@material-ui/core';
import { log } from '../util/errors';
import { apiGetServers, Server } from '../util/api';
import { MsgType, SayEvt, ServerLog } from '../util/game_events';
import useWebSocket, { ReadyState } from 'react-use-websocket';

export const ServerLogView = (): JSX.Element => {
    const proto = location.protocol === 'https:' ? 'wss' : 'ws';
    const port = location.port ? ':' + location.port : '';
    const messageHistory = useRef<ServerLog[]>([]);
    const [prev, setPrev] = useState<string>('');
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
        if (lastJsonMessage != prev || !messageHistory.current) {
            setPrev(lastJsonMessage);
            messageHistory.current.push(lastJsonMessage);
        }
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
    // eslint-disable-next-line
    // @ts-ignore
    const [serverIDs, setServerIDs] = useState<number[]>([]);
    // const [entries, setEntries] = useState<ServerLog[]>([]);
    // eslint-disable-next-line
    // @ts-ignore
    const [renderLimit, setRenderLimit] = useState<number>(10000);
    // eslint-disable-next-line
    // @ts-ignore
    const [filterServerIDs, setFilterServerIDs] = useState<number[]>([]);
    // const [filterMsgTypes, setFilterMsgTypes] = useState<MsgType[]>([
    //     MsgType.Any
    // ]);
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

    useEffect(() => {
        setServerIDs([1, 2]);
    }, []);
    useEffect(() => {
        setFilterServerIDs([1]);
    }, []);

    return (
        <Grid container>
            <Grid item xs={6}>
                <select
                    onChange={(event: SyntheticEvent) => {
                        setRenderLimit(
                            parseInt((event.target as HTMLSelectElement).value)
                        );
                    }}
                >
                    <option value={100}>100</option>
                    <option value={500}>500</option>
                    <option value={1000}>1000</option>
                    <option value={10000}>10000</option>
                    <option value={Number.MAX_SAFE_INTEGER}>inf.</option>
                </select>
            </Grid>
            <Grid item xs={6}>
                <select>
                    {servers &&
                        servers.map((value) => (
                            <option
                                key={`srv-${value.server_id}`}
                                value={value.server_id}
                            >
                                {value.server_name}
                            </option>
                        ))}
                </select>
            </Grid>
            <Grid item xs={12}>
                <h5>
                    Status: {readyState} {connectionStatus}
                </h5>
            </Grid>
            <Grid item xs={12}>
                <Grid container>
                    {messageHistory.current
                        .filter((value) => value)
                        .map((msg, i) => renderServerLog(msg, i))}
                </Grid>
            </Grid>
        </Grid>
    );
};

export const renderServerLog = (l: ServerLog, i: number): JSX.Element => {
    switch (l.event_type) {
        case MsgType.Say: {
            const e = l.payload as SayEvt;
            return (
                <Grid key={`sl-${i}`} item xs={12}>
                    {e.msg}
                </Grid>
            );
        }
        default:
            return <div key={`sl-${i}`} />;
    }
};
