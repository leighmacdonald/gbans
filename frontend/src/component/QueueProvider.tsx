import { ReactNode, useCallback, useEffect, useState } from 'react';
import useWebSocket, { ReadyState } from 'react-use-websocket';
import { useModal } from '@ebay/nice-modal-react';
import { uuidv7 } from 'uuidv7';
import {
    ClientStatePayload,
    defaultAvatarHash,
    GameStartPayload,
    JoinQueuePayload,
    LeaveQueuePayload,
    Operation,
    PermissionLevel,
    pingPayload,
    PurgePayload,
    QueueMember,
    QueuePayload,
    ServerQueueMessage,
    ServerQueueState,
    websocketURL
} from '../api';
import { QueueCtx } from '../hooks/useQueueCtx';
import * as killsound from '../icons/Killsound.mp3';
import { readAccessToken } from '../util/auth/readAccessToken.ts';
import { logErr } from '../util/errors.ts';
import { transformCreatedOnDate } from '../util/time.ts';
import { ModalQueueJoin } from './modal';

export const QueueProvider = ({ children }: { children: ReactNode }) => {
    const [isReady, setIsReady] = useState(false);
    const [users, setUsers] = useState<QueueMember[]>([]);
    const [messages, setMessages] = useState<ServerQueueMessage[]>([]);
    const [showChat, setShowChat] = useState(false);
    const [servers, setServers] = useState<ServerQueueState[]>([]);
    const [lastPong, setLastPong] = useState(new Date());

    const modal = useModal(ModalQueueJoin);

    const { readyState, sendJsonMessage, lastJsonMessage } = useWebSocket(websocketURL(), {
        queryParams: { token: readAccessToken() },
        share: false,
        // heartbeat: true,
        reconnectInterval: 10,
        shouldReconnect: () => true
    });

    useEffect(() => {
        switch (readyState) {
            case ReadyState.OPEN:
                setIsReady(true);
                sendJsonMessage({ op: Operation.Ping, payload: { created_on: new Date() } } as pingPayload);
                setMessages((prevState) => [
                    ...prevState,
                    {
                        created_on: new Date(),
                        body_md: 'Connected to queue',
                        avatarhash: '',
                        permission_level: PermissionLevel.Reserved,
                        personaname: 'SYSTEM',
                        steam_id: 'SYSTEM',
                        message_id: uuidv7()
                    } as ServerQueueMessage
                ]);
                break;
            case ReadyState.CLOSED:
                setIsReady(false);
                setMessages((prevState) => [
                    ...prevState,
                    {
                        created_on: new Date(),
                        body_md: 'Disconnected from queue',
                        avatarhash: '',
                        permission_level: PermissionLevel.Reserved,
                        personaname: 'SYSTEM',
                        steam_id: 'SYSTEM',
                        message_id: uuidv7()
                    } as ServerQueueMessage
                ]);
                break;
            default:
                break;
        }
    }, [readyState]);

    useEffect(() => {
        const request = lastJsonMessage as QueuePayload<never>;
        if (!request) {
            return;
        }
        handleIncomingOperation(request).catch(logErr);
    }, [lastJsonMessage]);

    const handleIncomingOperation = async (request: QueuePayload<never>) => {
        switch (request.op) {
            case Operation.Pong: {
                setLastPong(new Date());
                break;
            }

            case Operation.StateUpdate: {
                updateState(request.payload as ClientStatePayload);
                break;
            }

            case Operation.MessageRecv: {
                setMessages((prev) => [...prev, transformCreatedOnDate(request.payload as ServerQueueMessage)]);
                break;
            }

            case Operation.StartGame: {
                await startGame(request.payload as GameStartPayload);
                break;
            }

            case Operation.Purge: {
                purgeMessages((request.payload as PurgePayload).message_ids);
                break;
            }
        }
    };

    const updateState = (state: ClientStatePayload) => {
        if (state.update_users) {
            setUsers(state.users);
        }
        if (state.update_servers) {
            setServers(state.servers);
        }
    };

    const purgeMessages = (message_ids: string[]) => {
        setMessages((prevState) => prevState.filter((m) => !message_ids.includes(m.message_id)));
    };

    const startGame = async (gameStart: GameStartPayload) => {
        const audio = new Audio(killsound.default);
        try {
            await audio.play();
        } catch (e) {
            logErr(e);
        }

        await modal.show({ gameStart });
    };

    const sendMessage = useCallback((message: string) => {
        sendJsonMessage<QueuePayload<ServerQueueMessage>>({
            op: Operation.MessageSend,
            payload: {
                body_md: message,
                personaname: 'queue',
                permission_level: PermissionLevel.Reserved,
                avatarhash: defaultAvatarHash,
                steam_id: 'queue',
                created_on: new Date(),
                message_id: uuidv7()
            }
        });
    }, []);

    const joinQueue = (servers: string[]) => {
        sendJsonMessage<QueuePayload<JoinQueuePayload>>({
            op: Operation.JoinQueue,
            payload: {
                servers: servers.map(Number)
            }
        });
    };
    const leaveQueue = (servers: string[]) => {
        sendJsonMessage<QueuePayload<LeaveQueuePayload>>({
            op: Operation.LeaveQueue,
            payload: {
                servers: servers.map(Number)
            }
        });
    };

    return (
        <QueueCtx.Provider
            value={{
                users,
                servers,
                messages,
                isReady,
                sendMessage,
                joinQueue,
                leaveQueue,
                lastPong,
                showChat,
                setShowChat
            }}
        >
            {children}
        </QueueCtx.Provider>
    );
};
