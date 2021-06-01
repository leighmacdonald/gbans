import React, { useEffect, useMemo, useRef, useState } from 'react';
import SteamID from 'steamid';
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
import {
    apiGetServers,
    encode,
    PayloadType,
    Person,
    Server,
    WebSocketPayload
} from '../util/api';
import {
    eventNames,
    LogEvent,
    MsgType,
    Pos,
    StringIsNumber
} from '../util/game_events';
import useWebSocket, { ReadyState } from 'react-use-websocket';
import { takeRight } from 'lodash-es';
import { parseDateTime } from '../util/text';
import format from 'date-fns/format';

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
    const maxCacheSize = 10000;
    const classes = useStyles();
    const proto = location.protocol === 'https:' ? 'wss' : 'ws';
    const port = location.port ? ':' + location.port : '';
    // TODO Upper limit for how many messages well store
    const messageHistory = useRef<LogEvent[]>([]);
    const [filterSteamID, setFilterSteamID] = useState<SteamID>(
        new SteamID('')
    );
    const [filteredMessages, setFilteredMessages] = useState<LogEvent[]>([]);
    const { sendJsonMessage, lastJsonMessage, readyState } = useWebSocket(
        // `${proto}://gbans.uncledane.com/ws`
        `${proto}://${location.host}${port}/ws`,
        {
            onOpen: () => {
                sendJsonMessage(
                    encode(PayloadType.authType, {
                        token: localStorage.getItem('token')
                    })
                );
            },
            //Will attempt to reconnect on all close events, such as server shutting down
            shouldReconnect: () => true
        }
    );

    messageHistory.current = useMemo(() => {
        if (!lastJsonMessage) {
            return messageHistory.current;
        }
        const p = lastJsonMessage as WebSocketPayload;
        switch (p.payload_type) {
            case PayloadType.logType:
                messageHistory.current.push(
                    (p as WebSocketPayload<LogEvent>).data
                );
                if (messageHistory.current.length >= maxCacheSize) {
                    messageHistory.current.shift();
                }
                break;
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
    const [renderLimit, setRenderLimit] = useState<number>(25);
    const [filterServerIDs, setFilterServerIDs] = useState<number[]>([]);
    const [filterMsgTypes, setFilterMsgTypes] = useState<MsgType[]>([
        MsgType.Say,
        MsgType.SayTeam,
        MsgType.WRoundWin,
        MsgType.ShotFired,
        MsgType.Killed,
        MsgType.Connected,
        MsgType.Disconnected
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

    const handleChangeFilterMsg = (
        event: React.ChangeEvent<HTMLSelectElement> | any
    ) => {
        const v = event.target.value.filter(StringIsNumber);
        setFilterMsgTypes(v);
    };

    const handleChangeServers = (
        event: React.ChangeEvent<HTMLSelectElement> | any
    ) => {
        setFilterServerIDs(event.target.value);
    };
    const handleChangeRenderLimit = (
        event: React.ChangeEvent<HTMLSelectElement> | any
    ) => {
        setRenderLimit(parseInt(event.target.value));
    };

    const onFilterSteamIDChange = (
        event: React.ChangeEvent<HTMLInputElement>
    ) => {
        setFilterSteamID(new SteamID(event.target.value));
    };

    useEffect(() => {
        let logs = messageHistory.current.filter((v) => v);
        if (filterServerIDs.length > 0) {
            logs = logs.filter((s) =>
                filterServerIDs.includes(s.server.server_id)
            );
        }
        if (filterSteamID.isValid()) {
            logs = logs.filter(
                (s) =>
                    s.player1?.steam_id == filterSteamID.getSteamID64() ||
                    s.player2?.steam_id == filterSteamID.getSteamID64()
            );
        }
        if (
            filterMsgTypes.length > 0 &&
            !filterMsgTypes.includes(MsgType.Any)
        ) {
            logs = logs.filter((s) => filterMsgTypes.includes(s.event_type));
        }
        logs = takeRight<LogEvent>(logs, renderLimit);
        setFilteredMessages(logs);
    }, [
        setFilterServerIDs,
        setFilterMsgTypes,
        setRenderLimit,
        lastJsonMessage,
        filterServerIDs,
        filterSteamID,
        filterMsgTypes,
        renderLimit
    ]);

    return (
        <Grid container>
            <Grid item xs={3}>
                <FormControl className={classes.formControl} fullWidth>
                    <TextField
                        onChange={onFilterSteamIDChange}
                        id="standard-basic"
                        label="SteamID"
                    />
                </FormControl>
            </Grid>
            <Grid item xs={2}>
                <FormControl className={classes.formControl} fullWidth>
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
            <Grid item xs={7}>
                <FormControl className={classes.formControl} fullWidth>
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
                <FormControl className={classes.formControl} fullWidth>
                    <InputLabel id="msg-filters-label">
                        Message Filters
                    </InputLabel>
                    <Select
                        labelId="msg-filters-label"
                        id="msg-filters"
                        multiple
                        value={filterMsgTypes}
                        onChange={handleChangeFilterMsg}
                        defaultValue={filterMsgTypes}
                        input={<Input />}
                    >
                        {Object.values(MsgType)
                            .filter(StringIsNumber)
                            .map((mt) => (
                                <MenuItem key={mt} value={mt}>
                                    {MsgType[mt as MsgType]}
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

const renderPersonColumn = (p: Person | undefined): JSX.Element => {
    let name = '';
    if (p?.personaname) {
        name = p.personaname;
    } else if (p?.steam_id) {
        name = p.steam_id;
    }
    return <a style={{ fontWeight: 700 }}>{name}</a>;
};

const renderServerColumn = (s: Server): JSX.Element => {
    return <a style={{ fontWeight: 700 }}>{s.server_name}</a>;
};

const renderEventTypeColumn = (t: MsgType): JSX.Element => {
    return <a style={{ fontWeight: 700 }}>{eventNames[t]}</a>;
};

const renderEventTimeColumn = (t: string): JSX.Element => {
    return (
        <a style={{ fontWeight: 700 }}>{format(parseDateTime(t), 'HH:mm')}</a>
    );
};

const renderPosColumn = (p: Pos): JSX.Element => {
    return (
        <span>
            {p.x}, {p.y}, {p.z}
        </span>
    );
};

export const renderServerLog = (l: LogEvent, i: number): JSX.Element => {
    if (!l) {
        return <></>;
    }
    let v = <></>;
    switch (l.event_type) {
        case MsgType.UnhandledMsg: {
            break;
        }
        case MsgType.UnknownMsg: {
            v = <div>{JSON.stringify(l.event)}</div>;
            break;
        }
        case MsgType.Killed: {
            v = (
                <div>
                    Weapon: <b>{l.event['weapon']}</b>
                </div>
            );
            break;
        }
        case MsgType.KillAssist: {
            v = <div>Kill assist</div>;
            break;
        }
        case MsgType.Suicide: {
            v = (
                <div>
                    Suicided (pos:{' '}
                    <b>{renderPosColumn(l.event['pos'] as Pos)}</b>)
                </div>
            );
            break;
        }
        case MsgType.ShotFired: {
            v = (
                <div>
                    <b>{l.event['weapon']}</b>
                </div>
            );
            break;
        }
        case MsgType.ShotHit: {
            v = (
                <div>
                    <b>{l.event['weapon']}</b>
                </div>
            );
            break;
        }
        case MsgType.Domination: {
            v = <div>Dominated</div>;
            break;
        }
        case MsgType.Revenge: {
            v = <div>Got revenge</div>;
            break;
        }
        case MsgType.Pickup: {
            v = <div>{l.event['pickup']}</div>;
            break;
        }
        case MsgType.EmptyUber: {
            v = <div>Uber empty</div>;
            break;
        }
        case MsgType.MedicDeath: {
            v = (
                <div>
                    Medic death (uber: {l.event['uber']}) (healing:{' '}
                    {l.event['healing']})
                </div>
            );
            break;
        }
        case MsgType.MedicDeathEx: {
            v = <div>Medic death (pct: {l.event['uber_pct']})</div>;
            break;
        }
        case MsgType.LostUberAdv: {
            v = <div>Uber advantage lost ({l.event['advtime']}s)</div>;
            break;
        }
        case MsgType.ChargeReady: {
            v = (
                <div>
                    Uber <b>ready</b>
                </div>
            );
            break;
        }
        case MsgType.ChargeDeployed: {
            v = (
                <div>
                    Charge deployed (<b>{l.event['medigun']}</b>)
                </div>
            );
            break;
        }
        case MsgType.ChargeEnded: {
            v = (
                <div>
                    Charge ended (duration: <b>{l.event['duration']}s</b>)
                </div>
            );
            break;
        }
        case MsgType.Healed: {
            v = (
                <div>
                    <b>{l.event['healing']}</b>
                </div>
            );
            break;
        }
        case MsgType.Extinguished: {
            v = (
                <div>
                    With <b>{l.event['weapon']}</b>
                </div>
            );
            break;
        }
        case MsgType.BuiltObject: {
            v = (
                <div>
                    Built <b>{l.event['object']}</b>
                </div>
            );
            break;
        }
        case MsgType.CarryObject: {
            v = (
                <div>
                    Carried <b>{l.event['object']}</b>
                </div>
            );
            break;
        }
        case MsgType.KilledObject: {
            v = (
                <div>
                    Destroyed <b>{l.event['object']}</b> with{' '}
                    <b>{l.event['weapon']}</b>
                </div>
            );
            break;
        }
        case MsgType.DetonatedObject: {
            v = (
                <div>
                    Detonated <b>{l.event['object']}</b>
                </div>
            );
            break;
        }
        case MsgType.DropObject: {
            v = (
                <div>
                    Dropped <b>{l.event['object']}</b>
                </div>
            );
            break;
        }
        case MsgType.FirstHealAfterSpawn: {
            v = (
                <div>
                    First heal took <b>{l.event['time']}s</b>
                </div>
            );
            break;
        }
        case MsgType.CaptureBlocked: {
            v = (
                <div>
                    Capture <b>Blocked</b> <b>{l.event['cp_name']}</b> (
                    <b>{l.event['cp']}</b>
                </div>
            );
            break;
        }
        case MsgType.KilledCustom: {
            v = <div>custom_kill {l.event['custom_kill']}</div>;
            break;
        }
        case MsgType.PointCaptured: {
            v = (
                <div>
                    Team <b>{l.event['team']}</b>
                    CP <b>{l.event['cp_name']}</b> (<b>{l.event['cp']}</b>) Num{' '}
                    Num <b>{l.event['num_cappers']}</b>
                    <b>{l.event['body']}</b>
                </div>
            );
            break;
        }
        case MsgType.JoinedTeam: {
            v = (
                <div>
                    <b>{l.event['team']}</b>
                </div>
            );
            break;
        }
        case MsgType.ChangeClass: {
            v = (
                <div>
                    <b>{l.event['class']}</b>
                </div>
            );
            break;
        }
        case MsgType.SpawnedAs: {
            v = (
                <div>
                    <b>{l.event['class']}</b>
                </div>
            );
            break;
        }
        case MsgType.WRoundOvertime: {
            v = (
                <div>
                    Round <b>Overtime</b>
                </div>
            );
            break;
        }
        case MsgType.WRoundStart: {
            v = (
                <div>
                    Round <b>Start</b>
                </div>
            );
            break;
        }
        case MsgType.WRoundWin: {
            v = <div>Round win {l.event['winner']}</div>;
            break;
        }
        case MsgType.WRoundLen: {
            v = (
                <div>
                    Round length <b>{l.event['length']}</b>
                </div>
            );
            break;
        }
        case MsgType.WTeamScore: {
            v = (
                <div>
                    {l.event['team']} Score {l.event['score']} Players{' '}
                    {l.event['players']}
                </div>
            );
            break;
        }
        case MsgType.WTeamFinalScore: {
            v = (
                <div>
                    Score {l.event['score']} Players {l.event['players']}
                </div>
            );
            break;
        }
        case MsgType.WGameOver: {
            v = <div>Game Over (reason: {l.event['reason']})</div>;
            break;
        }
        case MsgType.WPaused: {
            v = (
                <div>
                    Game <b>Paused</b>
                </div>
            );
            break;
        }
        case MsgType.WResumed: {
            v = (
                <div>
                    Game <b>Resumed</b>
                </div>
            );
            break;
        }
        case MsgType.CVAR: {
            v = (
                <div>
                    CVAR <b>{l.event['cvar']}</b> &gt; <b>{l.event['value']}</b>
                </div>
            );
            break;
        }
        case MsgType.Connected: {
            v = <div>Connected</div>;
            break;
        }
        case MsgType.Disconnected: {
            v = <div>Disconnected</div>;
            break;
        }
        case MsgType.Entered: {
            v = <div>Entered</div>;
            break;
        }
        case MsgType.Validated: {
            v = <div>Validated</div>;
            break;
        }
        case MsgType.RCON: {
            v = <div>RCON {JSON.stringify(l.event)}</div>;
            break;
        }
        case MsgType.LogStart: {
            v = <div>Map loaded</div>;
            break;
        }
        case MsgType.LogStop: {
            v = <div>Map unloaded</div>;
            break;
        }
        case MsgType.Damage: {
            let rd = <></>;
            if (l.event['realdamage']) {
                rd = (
                    <span>
                        {' (real: '} <b>{l.event['realdamage']}</b>
                        {')'}
                    </span>
                );
            }
            v = (
                <div>
                    {'(damage: '} <b>{l.event['damage']}</b>){rd}
                </div>
            );
            break;
        }
        case MsgType.SayTeam: {
            v = (
                <div>
                    <b>
                        {'(team) '}
                        {l.event['msg']}
                    </b>
                </div>
            );
            break;
        }
        case MsgType.Say: {
            v = (
                <div>
                    <b>{l.event['msg']}</b>
                </div>
            );
            break;
        }
        default: {
            v = <div>{JSON.stringify(l.event)}</div>;
        }
    }
    let bg = 'inherit';
    if (l.event['team']) {
        if (l.event['team'] === 'Red') {
            bg = 'rgba(139,12,12,0.25)';
        } else if (l.event['team'] === 'Blue') {
            bg = 'rgba(12,48,139,0.25)';
        }
    }

    return (
        <Grid key={`sl-${i}`} item xs={12}>
            <Grid container style={{ backgroundColor: bg }}>
                <Grid item xs={1}>
                    {renderEventTimeColumn(l.created_on)}
                </Grid>
                <Grid item xs={1}>
                    {renderServerColumn(l.server)}
                </Grid>
                <Grid item xs={2}>
                    {renderPersonColumn(l.player1)}
                </Grid>
                <Grid item xs={2}>
                    {renderPersonColumn(l.player2)}
                </Grid>
                <Grid item xs={1}>
                    {renderEventTypeColumn(l.event_type)}
                </Grid>
                <Grid item xs={5}>
                    {v}
                </Grid>
            </Grid>
        </Grid>
    );
};
