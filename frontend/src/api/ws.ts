import { JsonObject } from 'react-use-websocket/dist/lib/types';

export enum MsgType {
    // Pug
    wsMsgTypePugCreateLobbyRequest = 1000,
    wsMsgTypePugCreateLobbyResponse = 1001,
    wsMsgTypePugLeaveLobbyRequest = 1002,
    wsMsgTypePugLeaveLobbyResponse = 1003,
    wsMsgTypePugJoinLobbyRequest = 1004,
    wsMsgTypePugJoinLobbyResponse = 1005,
    wsMsgTypePugUserMessageRequest = 1006,
    wsMsgTypePugUserMessageResponse = 1007,
    wsMsgTypePugLobbyListStatesRequest = 1008,
    wsMsgTypePugLobbyListStatesResponse = 1009,

    // Quickplay
    wsMsgTypeQPCreateLobbyRequest = 2000,
    wsMsgTypeQPCreateLobbyResponse = 2001,
    wsMsgTypeQPLeaveLobbyRequest = 2002,
    wsMsgTypeQPLeaveLobbyResponse = 2003,
    wsMsgTypeQPJoinLobbyRequest = 2004,
    wsMsgTypeQPJoinLobbyResponse = 2005,
    wsMsgTypeQPUserMessageRequest = 2006,
    wsMsgTypeQPUserMessageResponse = 2007
}

// All websocket messages must be wrapped in this payload container
export interface wsValue<T extends JsonObject> extends JsonObject {
    msg_type: MsgType;
    status: boolean;
    payload: T;
}

export const encode = <T>(msg_type: MsgType, payload: T, status?: boolean) => {
    return { msg_type, payload, status: status == undefined ? true : status };
};
