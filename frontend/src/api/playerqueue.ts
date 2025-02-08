import { apiCall, PermissionLevel } from './common.ts';

export enum Operation {
    Ping,
    Pong,
    JoinQueue,
    LeaveQueue,
    Message,
    StateUpdate,
    StartGame,
    Purge,
    Bye,
    ChatStatusChange
}

export type QueueMember = {
    name: string;
    steam_id: string;
    hash: string;
};

export type QueueRequest<T> = {
    op: Operation;
    payload: T;
};

export type PurgePayload = {
    message_ids: number[];
};

export type clientQueueState = {
    steam_id: string;
};
export type ServerQueueState = {
    server_id: number;
    members: clientQueueState[];
};

export type createMessage = {
    body_md: string;
};

export type ServerQueueMessage = {
    steam_id: string;
    created_on: Date;
    personaname: string;
    avatarhash: string;
    permission_level: PermissionLevel;
    body_md: string;
    message_id: number;
};

export type JoinQueuePayload = {
    servers: number[];
};

export type LeaveQueuePayload = JoinQueuePayload;

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

export type QueueServer = {
    name: string;
    short_name: string;
    cc: string;
    connect_url: string;
    connect_command: string;
};

export type ChatStatusChangePayload = {
    status: ChatStatus;
    reason: string;
};

export type GameStartPayload = {
    users: Member[];
    server: QueueServer;
};

export type ChatStatus = 'readwrite' | 'readonly' | 'noaccess';

export const websocketURL = () => {
    let protocol = 'ws';
    if (location.protocol === 'https:') {
        protocol = 'wss:';
    }
    return `${protocol}://${location.host}/ws`;
};

export const apiQueueMessagesDelete = async (message_id: number, count: number) => {
    return await apiCall(`/api/playerqueue/messages/${message_id}/${count}`, 'DELETE', {});
};

export const apiQueueSetUserStatus = async (steam_id: string, chat_status: ChatStatus, reason: string) => {
    return await apiCall(`/api/playerqueue/status/${steam_id}`, 'PUT', { chat_status, reason });
};
