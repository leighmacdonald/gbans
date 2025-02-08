import { ReactNode, useEffect, useState } from 'react';
import useWebSocket, { ReadyState } from 'react-use-websocket';
import { useModal } from '@ebay/nice-modal-react';
import {
    ChatStatus,
    ChatStatusChangePayload,
    ClientStatePayload,
    createMessage,
    GameStartPayload,
    JoinQueuePayload,
    LeaveQueuePayload,
    Operation,
    PermissionLevel,
    pingPayload,
    PurgePayload,
    QueueMember,
    QueueRequest,
    ServerQueueMessage,
    ServerQueueState,
    websocketURL
} from '../api';
import { useAuth } from '../hooks/useAuth.ts';
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
    const { profile } = useAuth();
    const [chatStatus, setChatStatus] = useState<ChatStatus>(profile.playerqueue_chat_status);
    const [reason, setReason] = useState<string>('');

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
                        steam_id: 'SYSTEM'
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
                        steam_id: 'SYSTEM'
                    } as ServerQueueMessage
                ]);
                break;
            default:
                break;
        }
    }, [readyState]);

    useEffect(() => {
        const request = lastJsonMessage as QueueRequest<never>;
        if (!request) {
            return;
        }
        handleIncomingOperation(request).catch(logErr);
    }, [lastJsonMessage]);

    const handleIncomingOperation = async (request: QueueRequest<never>) => {
        switch (request.op) {
            case Operation.Pong: {
                setLastPong(new Date());
                break;
            }

            case Operation.StateUpdate: {
                updateState(request.payload as ClientStatePayload);
                break;
            }

            case Operation.Message: {
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
            case Operation.ChatStatusChange: {
                const pl = request.payload as ChatStatusChangePayload;
                setChatStatus(pl.status);
                if (pl.status == 'noaccess') {
                    setMessages([]);
                }
                setReason(pl.reason);
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

    const purgeMessages = (message_ids: number[]) => {
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

    const sendMessage = (body_md: string) => {
        sendJsonMessage<QueueRequest<createMessage>>({
            op: Operation.Message,
            payload: {
                body_md
            }
        });
    };

    const joinQueue = (servers: string[]) => {
        sendJsonMessage<QueueRequest<JoinQueuePayload>>({
            op: Operation.JoinQueue,
            payload: {
                servers: servers.map(Number)
            }
        });
    };
    const leaveQueue = (servers: string[]) => {
        sendJsonMessage<QueueRequest<LeaveQueuePayload>>({
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
                setShowChat,
                chatStatus,
                reason
            }}
        >
            {children}
        </QueueCtx.Provider>
    );
};
