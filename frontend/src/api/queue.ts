import { readAccessToken } from '../util/auth/readAccessToken.ts';
import { PermissionLevel } from './common.ts';

export enum Operation {
    Ping,
    Pong,
    Join,
    Leave,
    MessageSend,
    MessageRecv
}

export type queuePayload<T> = {
    op: Operation;
    payload: T;
};

export type pingPayload = queuePayload<{ created_on: Date }>;

export type ServerQueueMessage = {
    steam_id: string;
    created_on: Date;
    personaname: string;
    avatarhash: string;
    permission_level: PermissionLevel;
    body_md: string;
    id: string;
};

export const websocketURL = () => {
    let protocol = 'ws';
    if (location.protocol === 'https:') {
        protocol = 'wss:';
    }
    const token = readAccessToken();
    return `${protocol}://${location.host}/ws?token=${token}`;
};

export const isOperationType = (message: WebSocketEventMap['message'], opType: Operation) => {
    return (JSON.parse(message.data) as queuePayload<never>).op === opType;
};
