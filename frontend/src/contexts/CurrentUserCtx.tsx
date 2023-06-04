import { createContext, useContext } from 'react';
import {
    PermissionLevel,
    readAccessToken,
    userKey,
    UserProfile,
    writeAccessToken,
    writeRefreshToken
} from '../api';
import SteamID from 'steamid';

export const GuestProfile: UserProfile = {
    updated_on: new Date(),
    created_on: new Date(),
    permission_level: PermissionLevel.Guest,
    discord_id: '',
    avatar: '',
    avatarfull: '',
    steam_id: new SteamID(''),
    ban_id: 0,
    name: 'Guest',
    muted: false
};

export type CurrentUser = {
    currentUser: UserProfile;
    setCurrentUser: (profile: UserProfile) => void;
    getToken: () => string;
    setToken: (token: string) => void;
    getRefreshToken: () => string;
    setRefreshToken: (token: string) => void;
};

export const CurrentUserCtx = createContext<CurrentUser>({
    currentUser: GuestProfile,
    setCurrentUser: (profile: UserProfile) => {
        try {
            localStorage.setItem(userKey, JSON.stringify(profile));
        } catch (e) {
            return;
        }
    },
    getToken: readAccessToken,
    setToken: writeAccessToken,
    getRefreshToken: readAccessToken,
    setRefreshToken: writeRefreshToken
});

// eslint-disable-next-line @typescript-eslint/explicit-module-boundary-types
export const useCurrentUserCtx = () => useContext(CurrentUserCtx);
