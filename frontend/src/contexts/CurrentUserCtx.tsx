import { createContext } from 'react';
import { PermissionLevel, userKey, UserProfile } from '../api';
import { GuestProfile } from '../util/profile.ts';

export type CurrentUser = {
    currentUser: UserProfile;
    setCurrentUser: (profile: UserProfile) => void;
    isAuthenticated: boolean;
    permissionLevel: PermissionLevel;
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
    isAuthenticated: false,
    permissionLevel: PermissionLevel.Guest
});
