import { apiCall, PermissionLevel } from './common.ts';

export const apiQueueMessagesDelete = async (message_id: number, count: number) => {
    return await apiCall(`/api/playerqueue/messages/${message_id}/${count}`, 'DELETE', {});
};

export const apiQueueSetUserStatus = async (steam_id: string, chat_status: ChatStatus, reason: string) => {
    return await apiCall(`/api/playerqueue/status/${steam_id}`, 'PUT', { chat_status, reason });
};

export enum Operation {
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

export type ClientQueueState = {
    steam_id: string;
};

export type LobbyState = {
    server_id: number;
    members: ClientQueueState[];
};

export type MessageCreatePayload = {
    body_md: string;
};

export type ChatLog = {
    steam_id: string;
    created_on: Date;
    personaname: string;
    avatarhash: string;
    permission_level: PermissionLevel;
    body_md: string;
    message_id: number;
};

export type MessagePayload = {
    messages: ChatLog[];
};

export type JoinPayload = {
    servers: number[];
};

export type LeavePayload = JoinPayload;

export type Member = {
    name: string;
    steam_id: string;
    hash: string;
};

export type ClientStatePayload = {
    update_users: boolean;
    update_servers: boolean;
    lobbies: LobbyState[];
    users: Member[];
};

export type LobbyServer = {
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
    server: LobbyServer;
};

export type ChatStatus = 'readwrite' | 'readonly' | 'noaccess';

export const websocketURL = () => {
    let protocol = 'ws';
    if (location.protocol === 'https:') {
        protocol = 'wss:';
    }
    return `${protocol}://${location.host}/ws`;
};
