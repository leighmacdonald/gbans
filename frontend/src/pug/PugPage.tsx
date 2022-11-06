import React, { useCallback, useEffect, useMemo, useState } from 'react';
import Grid from '@mui/material/Grid';
import Stack from '@mui/material/Stack';
import Avatar from '@mui/material/Avatar';
import Paper from '@mui/material/Paper';
import { Heading } from '../component/Heading';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import scoutIcon from '../icons/class_scout.png';
import soldierIcon from '../icons/class_soldier.png';
import pyroIcon from '../icons/class_pyro.png';
import demoIcon from '../icons/class_demoman.png';
import heavyIcon from '../icons/class_heavy.png';
import engyIcon from '../icons/class_engineer.png';
import medicIcon from '../icons/class_medic.png';
import sniperIcon from '../icons/class_sniper.png';
import spyIcon from '../icons/class_spy.png';
import { PugCtx } from '../contexts/PugCtx';
import { PugLobby, PugPlayer } from './pug';
import { Nullable } from '../util/types';
import { PlayerClassSelection } from './PlayerClassSelection';
import { JoinOrCreateLobby } from './JoinOrCreateLobby';
import {
    qpMsgType,
    qpRequestTypes,
    qpUserMessageI,
    readAccessToken
} from '../api';
import useWebSocket, { ReadyState } from 'react-use-websocket';

interface ClassBoxProps {
    src: string;
}

const ClassBox = ({ src }: ClassBoxProps) => {
    return (
        <Grid sx={{ height: 70 }} container>
            <Grid item xs={12} alignItems="center" alignContent={'center'}>
                <Avatar src={src} sx={{ textAlign: 'center' }} />
            </Grid>
        </Grid>
    );
};

export const PugPage = (): JSX.Element => {
    const { currentUser } = useCurrentUserCtx();

    const [lobby] = useState<Nullable<PugLobby>>(null);
    const [_, setMessageHistory] = useState<qpUserMessageI[]>([]);

    const token = useMemo(() => {
        return readAccessToken();
    }, []);

    const socketUrl = useMemo(() => {
        const parsedUrl = new URL(window.location.href);
        return `${parsedUrl.protocol == 'https' ? 'wss' : 'ws'}://${
            parsedUrl.host
        }/ws/pug`;
    }, []);

    const { readyState, lastJsonMessage /*, sendJsonMessage*/ } = useWebSocket(
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
            onClose: () => {
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
            onOpen: () => {
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

    const isReady = useMemo(() => {
        return readyState == ReadyState.OPEN;
    }, [readyState]);

    const joinLobby = useCallback(() => {
        if (!isReady) {
            return;
        }
        return;
    }, [isReady]);

    const leaveLobby = useCallback(
        (_: PugPlayer) => {
            if (!isReady) {
                return;
            }
            return;
        },
        [isReady]
    );

    const createLobby = useCallback(() => {
        if (!isReady) {
            return;
        }
        return;
    }, [isReady]);

    useEffect(() => {
        if (lastJsonMessage != null) {
            const p = lastJsonMessage as qpRequestTypes;
            switch (p.msg_type) {
                case qpMsgType.qpMsgTypeJoinLobbySuccess: {
                    // const req = p as qpMsgJoinedLobbySuccess;
                    // setLobby(req.payload.lobby);
                    return;
                }
                case qpMsgType.qpMsgTypeSendMsgRequest: {
                    // const req = p as qpUserMessage;
                    // setMessageHistory((prev) => prev.concat(req.payload));
                    return;
                }
                default: {
                    console.log(lastJsonMessage);
                }
            }
        }
    }, [lastJsonMessage]);

    return (
        <PugCtx.Provider value={{ createLobby, leaveLobby, joinLobby, lobby }}>
            <Grid container paddingTop={3} spacing={2}>
                {lobby ? (
                    <>
                        <Grid item xs={12}></Grid>
                        <Grid item xs={5}>
                            <Paper>
                                <Stack spacing={1}>
                                    <Heading>RED</Heading>
                                    <PlayerClassSelection
                                        reverse
                                        user={currentUser}
                                    />
                                    <PlayerClassSelection
                                        reverse
                                        user={currentUser}
                                    />
                                    <PlayerClassSelection reverse />
                                    <PlayerClassSelection
                                        reverse
                                        user={currentUser}
                                    />
                                    <PlayerClassSelection reverse />
                                    <PlayerClassSelection
                                        reverse
                                        user={currentUser}
                                    />
                                    <PlayerClassSelection reverse />
                                    <PlayerClassSelection
                                        reverse
                                        user={currentUser}
                                    />
                                    <PlayerClassSelection
                                        reverse
                                        user={currentUser}
                                    />
                                </Stack>
                            </Paper>
                        </Grid>
                        <Grid item xs={2}>
                            <Stack spacing={1}>
                                <Heading>Class</Heading>
                                <ClassBox src={scoutIcon} />
                                <ClassBox src={soldierIcon} />
                                <ClassBox src={pyroIcon} />
                                <ClassBox src={demoIcon} />
                                <ClassBox src={heavyIcon} />
                                <ClassBox src={engyIcon} />
                                <ClassBox src={medicIcon} />
                                <ClassBox src={sniperIcon} />
                                <ClassBox src={spyIcon} />
                            </Stack>
                        </Grid>
                        <Grid item xs={5}>
                            <Paper>
                                <Stack spacing={1}>
                                    <Heading bgColor={'#395c78'}>BLU</Heading>
                                    <PlayerClassSelection user={currentUser} />
                                    <PlayerClassSelection />
                                    <PlayerClassSelection user={currentUser} />
                                    <PlayerClassSelection user={currentUser} />
                                    <PlayerClassSelection user={currentUser} />
                                    <PlayerClassSelection />
                                    <PlayerClassSelection user={currentUser} />
                                    <PlayerClassSelection />
                                    <PlayerClassSelection user={currentUser} />
                                </Stack>
                            </Paper>
                        </Grid>
                    </>
                ) : (
                    <JoinOrCreateLobby />
                )}
            </Grid>
        </PugCtx.Provider>
    );
};
