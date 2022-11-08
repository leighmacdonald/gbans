import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { PugCtx } from '../contexts/PugCtx';
import { PugLobby, PugPlayer } from './pug';
import { Nullable } from '../util/types';

import {
    qpMsgType,
    qpRequestTypes,
    qpUserMessageI,
    readAccessToken
} from '../api';
import useWebSocket, { ReadyState } from 'react-use-websocket';
import { JoinOrCreateLobby } from './JoinOrCreateLobby';
import { PugLobbyView } from './PugLobbyView';

export const PugPage = (): JSX.Element => {
    const [lobby] = useState<Nullable<PugLobby>>(null);
    const [_, setMessageHistory] = useState<qpUserMessageI[]>([]);

    const token = useMemo(() => {
        return readAccessToken();
    }, []);

    const socketUrl = useMemo(() => {
        const parsedUrl = new URL(window.location.href);
        return `${parsedUrl.protocol == 'https' ? 'wss' : 'ws'}://${
            parsedUrl.host
        }/ws`;
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
            {lobby ? <PugLobbyView /> : <JoinOrCreateLobby />}
        </PugCtx.Provider>
    );
};
