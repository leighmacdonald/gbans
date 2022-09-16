import { JsonObject } from 'react-use-websocket/dist/lib/types';

export enum qpMsgType {
    qpMsgTypeJoin = 0,
    qpMsgTypeLeave,
    qpMsgTypeJoinLobby,
    qpMsgTypeSendMsg
}

export interface qpBaseQuery extends JsonObject {
    msg_type: qpMsgType;
    payload: JsonObject;
}

export interface qpUserMessage extends JsonObject {
    steam_id?: string;
    message: string;
    created_at: string;
}

interface qpClient extends JsonObject {
    leader: boolean;
    user: undefined;
}

export interface qpLobby extends JsonObject {
    lobby_id: string;
    clients: qpClient[];
}

export interface qpMsgJoinLobby extends JsonObject {
    lobby: qpLobby;
}
