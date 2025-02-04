import { PermissionLevel } from './common.ts';

export enum Operation {
    Ping,
    Pong,
    JoinQueue,
    LeaveQueue,
    MessageSend,
    MessageRecv,
    StateUpdate,
    StartGame,
    Purge
}

export type QueueMember = {
    name: string;
    steam_id: string;
    hash: string;
};

export type QueuePayload<T> = {
    op: Operation;
    payload: T;
};

export type PurgePayload = {
    message_ids: string[];
};

export type pingPayload = QueuePayload<{ created_on: Date }>;

export type clientQueueState = {
    steam_id: string;
};
export type ServerQueueState = {
    server_id: number;
    members: clientQueueState[];
};

export type ServerQueueMessage = {
    steam_id: string;
    created_on: Date;
    personaname: string;
    avatarhash: string;
    permission_level: PermissionLevel;
    body_md: string;
    message_id: string;
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

export type Server = {
    name: string;
    short_name: string;
    cc: string;
    connect_url: string;
    connect_command: string;
};

export type GameStartPayload = {
    users: Member[];
    server: Server;
};
