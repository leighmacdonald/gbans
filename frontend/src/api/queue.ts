import { PermissionLevel } from './common.ts';

export enum Operation {
    Ping,
    Pong,
    JoinQueue,
    LeaveQueue,
    MessageSend,
    MessageRecv,
    StateUpdate,
    StartGame
}

export type QueueMember = {
    name: string;
    steam_id: string;
    hash: string;
};

export type queuePayload<T> = {
    op: Operation;
    payload: T;
};

export type pingPayload = queuePayload<{ created_on: Date }>;

export type ServerQueueState = {
    server_id: number;
    members: string[];
};

export type ServerQueueMessage = {
    steam_id: string;
    created_on: Date;
    personaname: string;
    avatarhash: string;
    permission_level: PermissionLevel;
    body_md: string;
    id: string;
};

export type JoinQueuePayload = {
    servers: number[];
};

export type LeaveQueuePayload = JoinQueuePayload;

export const websocketURL = () => {
    let protocol = 'ws';
    if (location.protocol === 'https:') {
        protocol = 'wss:';
    }
    return `${protocol}://${location.host}/ws`;
};

export type Member = {
    name: string;
    steam_id: string;
    hash: string;
};

export type ClientStatePayload = {
    update_users: boolean;
    update_servers: boolean;
    servers: ServerQueueState[];
    users: Member[];
};
