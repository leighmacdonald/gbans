import { ReactNode, useCallback, useEffect, useState } from 'react';
import useWebSocket, { ReadyState } from 'react-use-websocket';
import { uuidv7 } from 'uuidv7';
import {
    ClientStatePayload,
    defaultAvatarHash,
    JoinQueuePayload,
    LeaveQueuePayload,
    Operation,
    PermissionLevel,
    pingPayload,
    QueueMember,
    queuePayload,
    ServerQueueMessage,
    ServerQueueState,
    websocketURL
} from '../api';
import { QueueCtx } from '../hooks/useQueueCtx';
import { readAccessToken } from '../util/auth/readAccessToken.ts';
import { transformCreatedOnDate } from '../util/time.ts';

export const QueueProvider = ({ children }: { children: ReactNode }) => {
    const [isReady, setIsReady] = useState(false);
    const [users, setUsers] = useState<QueueMember[]>([]);
    const [messages, setMessages] = useState<ServerQueueMessage[]>([]);
    const [showChat, setShowChat] = useState(false);
    const [servers, setServers] = useState<ServerQueueState[]>([]);
    const [lastPong, setLastPong] = useState(new Date());

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
                        id: uuidv7()
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
                        id: uuidv7()
                    } as ServerQueueMessage
                ]);
                break;
            default:
                break;
        }
    }, [readyState]);

    useEffect(() => {
        const request = lastJsonMessage as queuePayload<never>;
        if (!request) {
            return;
        }

        switch (request.op) {
            case Operation.Pong:
                setLastPong(new Date());
                break;
            case Operation.StateUpdate: {
                const payload = request.payload as ClientStatePayload;
                if (payload.update_users) {
                    setUsers(payload.users);
                }
                if (payload.update_servers) {
                    setServers(payload.servers);
                }
                break;
            }
            case Operation.MessageRecv: {
                const payload = request.payload as ServerQueueMessage;
                setMessages((prev) => [...prev, transformCreatedOnDate(payload)]);
                break;
            }

            case Operation.StartGame: {
                const payload = request.payload as ServerQueueState;
                startGame(payload);
            }
        }
    }, [lastJsonMessage]);

    const startGame = (state: ServerQueueState) => {
        alert(`start game: ${state.server_id}`);
    };

    const sendMessage = useCallback((message: string) => {
        sendJsonMessage<queuePayload<ServerQueueMessage>>({
            op: Operation.MessageSend,
            payload: {
                body_md: message,
                personaname: 'queue',
                permission_level: PermissionLevel.Reserved,
                avatarhash: defaultAvatarHash,
                steam_id: 'queue',
                created_on: new Date(),
                id: uuidv7()
            }
        });
    }, []);

    const joinQueue = (servers: string[]) => {
        sendJsonMessage<queuePayload<JoinQueuePayload>>({
            op: Operation.JoinQueue,
            payload: {
                servers: servers.map(Number)
            }
        });
    };
    const leaveQueue = (servers: string[]) => {
        sendJsonMessage<queuePayload<LeaveQueuePayload>>({
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
