import { UserProfile } from '../api';

export interface PugPlayer {
    person: UserProfile;
}

export interface PugLobby {
    lobbyId: string;
    players: PugPlayer[];
}
