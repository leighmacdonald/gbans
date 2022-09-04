import { createContext, useContext } from 'react';
import { PermissionLevel, tokenKey, userKey, UserProfile } from '../api';
import SteamID from 'steamid';
import { Nullable } from '../util/types';

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
    token: Nullable<string>;
    getToken: () => string;
    setToken: (token: string) => void;
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
    token: '',
    getToken: () => {
        try {
            return localStorage.getItem(tokenKey) as string;
        } catch (e) {
            return '';
        }
    },
    setToken: (userToken) => {
        localStorage.setItem(tokenKey, userToken);
    }
});

// eslint-disable-next-line @typescript-eslint/explicit-module-boundary-types
export const useCurrentUserCtx = () => useContext(CurrentUserCtx);
