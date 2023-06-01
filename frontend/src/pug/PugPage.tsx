import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { PugCtx } from './PugCtx';
import {
    PugLobby,
    wsMsgTypePugCreateLobbyRequest,
    wsMsgTypePugUserMessageRequest,
    wsMsgTypePugUserMessageResponse,
    wsPugResponseTypes
} from './pug';
import { Nullable } from '../util/types';

import { readAccessToken } from '../api';
import useWebSocket, { ReadyState } from 'react-use-websocket';
import { JoinOrCreateLobby } from './JoinOrCreateLobby';
import { PugLobbyView } from './PugLobbyView';
import { encode, MsgType, wsValue } from '../api/ws';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';

export const PugPage = () => {
    const [lobby, setLobby] = useState<Nullable<PugLobby>>(null);
    const [lobbies, setLobbies] = useState<PugLobby[]>([]);
    const [messages, setMessages] = useState<wsMsgTypePugUserMessageResponse[]>(
        []
    );
    const { sendFlash } = useUserFlashCtx();

    const socketUrl = useMemo(() => {
        const parsedUrl = new URL(window.location.href);
        return `${parsedUrl.protocol == 'https' ? 'wss' : 'ws'}://${
            parsedUrl.host
        }/ws`;
    }, []);

    const systemMsg = useCallback((message: string) => {
        setMessages((prevState) =>
            prevState.concat({
                message: message,
                created_at: new Date().toISOString()
            })
        );
    }, []);

    const { readyState, lastJsonMessage, sendJsonMessage } = useWebSocket(
        socketUrl,
        {
            onError: (event: WebSocketEventMap['error']) => {
                systemMsg(event.type);
            },
            onClose: () => {
                systemMsg('Lobby connection closed');
            },
            queryParams: {
                token: readAccessToken()
            },
            onOpen: () => {
                systemMsg('Lobby connection opened');
            }, //Will attempt to reconnect on all close events, such as server shutting down
            shouldReconnect: () => true
        }
    );

    const isReady = useMemo(() => {
        return readyState == ReadyState.OPEN;
    }, [readyState]);

    const joinLobby = useCallback(
        (lobby_id: string) => {
            if (!isReady) {
                return;
            }
            const msg = encode(MsgType.wsMsgTypePugJoinLobbyRequest, {
                lobby_id
            });
            sendJsonMessage(msg);
            return;
        },
        [isReady, sendJsonMessage]
    );

    const leaveLobby = useCallback(() => {
        if (!isReady) {
            return;
        }
        const msg = encode(MsgType.wsMsgTypePugLeaveLobbyRequest, {});
        sendJsonMessage(msg);
        return;
    }, [isReady, sendJsonMessage]);

    const createLobby = useCallback(
        async (opts: wsMsgTypePugCreateLobbyRequest) => {
            if (!isReady) {
                return;
            }
            const msg = encode(MsgType.wsMsgTypePugCreateLobbyRequest, opts);
            sendJsonMessage(msg);
            return;
        },
        [sendJsonMessage, isReady]
    );

    const sendMessage = useCallback(
        (body: string) => {
            if (!isReady) {
                return;
            }
            const msg = encode<wsMsgTypePugUserMessageRequest>(
                MsgType.wsMsgTypePugUserMessageRequest,
                {
                    message: body
                }
            );
            sendJsonMessage(msg);
        },
        [isReady, sendJsonMessage]
    );

    useEffect(() => {
        if (lastJsonMessage != null) {
            const last = lastJsonMessage as wsValue<wsPugResponseTypes>;
            switch (last.msg_type) {
                case MsgType.wsMsgTypePugCreateLobbyResponse: {
                    const lobby = last.payload.lobby as PugLobby;
                    setLobby(lobby);
                    break;
                }
                case MsgType.wsMsgTypePugJoinLobbyResponse: {
                    const lobby = last.payload.lobby as PugLobby;
                    setLobby(lobby);
                    break;
                }
                case MsgType.wsMsgTypePugLeaveLobbyResponse: {
                    setLobby(null);
                    break;
                }
                case MsgType.wsMsgTypePugLobbyListStatesResponse: {
                    const lobbies = last.payload.lobbies as PugLobby[];
                    setLobbies(lobbies);
                    console.log('state updated');
                    break;
                }
                case MsgType.wsMsgTypePugUserMessageResponse: {
                    const { payload } = last;
                    setMessages((prev) => {
                        if (!payload.message) {
                            return prev;
                        }
                        return [
                            ...prev,
                            payload as wsMsgTypePugUserMessageResponse
                        ];
                    });
                    return;
                }
                default: {
                    console.log(lastJsonMessage);
                }
            }
        }
    }, [lastJsonMessage, sendFlash]);

    return (
        <PugCtx.Provider
            value={{
                createLobby,
                leaveLobby,
                joinLobby,
                lobby,
                setLobby,
                sendMessage,
                messages,
                lobbies,
                setLobbies
            }}
        >
            {lobby ? <PugLobbyView /> : <JoinOrCreateLobby isReady={isReady} />}
        </PugCtx.Provider>
    );
};
