import { JsonObject } from 'react-use-websocket/dist/lib/types';

export enum qpMsgType {
    qpMsgTypeJoinLobbyRequest = 0,
    qpMsgTypeLeaveLobbyRequest,
    qpMsgTypeJoinLobbySuccess,
    qpMsgTypeSendMsgRequest
}

interface qpBaseMessage<T extends JsonObject> extends JsonObject {
    msg_type: qpMsgType;
    payload: T;
}

export interface qpUserMessageI extends JsonObject {
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

export interface qpMsgJoinedLobbySuccessI extends JsonObject {
    lobby: qpLobby;
}

export interface qpMsgJoinLobbyRequestI extends JsonObject {
    lobby_id: string;
}

export type qpUserMessage = qpBaseMessage<qpUserMessageI>;
export type qpMsgJoinLobbyRequest = qpBaseMessage<qpMsgJoinLobbyRequestI>;
export type qpMsgJoinedLobbySuccess = qpBaseMessage<qpMsgJoinedLobbySuccessI>;

export type qpRequestTypes =
    | qpMsgJoinLobbyRequest
    | qpUserMessage
    | qpMsgJoinedLobbySuccess;
