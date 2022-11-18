import { UserProfile } from '../api';
import { JsonObject } from 'react-use-websocket/dist/lib/types';
import { GameType } from '../component/formik/GameTypeField';
import { GameConfig } from '../component/formik/GameConfigField';
import { Nullable } from '../util/types';

export interface PugPlayer extends JsonObject {
    person: UserProfile & JsonObject;
}

export interface PugLobby extends JsonObject {
    lobbyId: string;
    leader: Nullable<PugPlayer>;
    clients: PugPlayer[];
    messages: wsMsgTypePugUserMessageResponse[];
    options: wsMsgTypePugCreateLobbyRequest;
}

export interface wsPugMsgCreateLobbyResponse extends JsonObject {
    lobby: PugLobby;
}

export interface wsPugMsgLobbyListStatesResponse extends JsonObject {
    lobbies: PugLobby[];
}

export interface wsMsgTypePugUserMessageRequest extends JsonObject {
    message: string;
}

export interface wsMsgTypePugUserMessageResponse extends JsonObject {
    user?: UserProfile & JsonObject;
    message: string;
    created_at: string;
}

export interface wsMsgTypePugCreateLobbyRequest extends JsonObject {
    game_type: GameType;
    game_config: GameConfig;
    map_name: string; // "map" conflicts too much
    description: string;
    discord_required: boolean;
    server_name: string;
}

export type wsPugRequestTypes =
    | wsMsgTypePugCreateLobbyRequest
    | wsMsgTypePugUserMessageRequest;

export type wsPugResponseTypes =
    | wsPugMsgCreateLobbyResponse
    | wsMsgTypePugUserMessageResponse
    | wsPugMsgLobbyListStatesResponse;
