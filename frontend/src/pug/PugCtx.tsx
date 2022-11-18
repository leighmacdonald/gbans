import { createContext, useContext } from 'react';
import { noop } from 'lodash-es';
import {
    PugLobby,
    wsMsgTypePugCreateLobbyRequest,
    wsMsgTypePugUserMessageResponse
} from './pug';
import { Nullable } from '../util/types';
import { GameConfig } from '../component/formik/GameConfigField';
import { GameType } from '../component/formik/GameTypeField';

export type PugContext = {
    lobby: Nullable<PugLobby>;
    lobbies: PugLobby[];
    setLobbies: (lobbies: PugLobby[]) => void;
    setLobby: (lobby: PugLobby) => void;
    joinLobby: (lobbyId: string) => void;
    leaveLobby: () => void;
    createLobby: (opts: wsMsgTypePugCreateLobbyRequest) => void;
    sendMessage: (body: string) => void;
    messages: wsMsgTypePugUserMessageResponse[];
};

export const PugCtx = createContext<PugContext>({
    lobbies: [],
    lobby: {
        lobbyId: '',
        players: [],
        clients: [],
        messages: [],
        leader: null,
        options: {
            description: '',
            discord_required: false,
            map_name: '',
            server_name: '',
            game_config: GameConfig.rgl,
            game_type: GameType.sixes
        }
    },
    setLobbies: () => noop,
    setLobby: () => noop,
    joinLobby: () => noop,
    leaveLobby: () => noop,
    createLobby: () => noop,
    sendMessage: () => noop,
    messages: []
});

export const usePugCtx = () => useContext(PugCtx);
