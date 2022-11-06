import { createContext, useContext } from 'react';
import { noop } from 'lodash-es';
import { PugLobby, PugPlayer } from '../pug/pug';
import { Team } from '../api';
import { Nullable } from '../util/types';

export type PugContext = {
    lobby: Nullable<PugLobby>;
    joinLobby: (player: PugPlayer, team: Team) => void;
    leaveLobby: (player: PugPlayer) => void;
    createLobby: () => void;
};

export const PugCtx = createContext<PugContext>({
    lobby: { lobbyId: '', players: [] },
    joinLobby: (_: PugPlayer, __: Team) => noop,
    leaveLobby: (_: PugPlayer) => noop,
    createLobby: () => noop
});

export const usePugCtx = () => useContext(PugCtx);
