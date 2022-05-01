import { createContext, useContext } from 'react';
import { UserProfile } from '../api';
import { noop } from 'lodash-es';

export const GuestProfile: UserProfile = {
    updated_on: new Date(),
    created_on: new Date(),
    permission_level: 0,
    discord_id: '',
    avatar: '',
    avatarfull: '',
    steam_id: '',
    ban_id: 0,
    name: 'Guest'
};

export type CurrentUser = {
    currentUser: UserProfile;
    setCurrentUser: (profile: UserProfile) => void;
};

export const CurrentUserCtx = createContext<CurrentUser>({
    currentUser: GuestProfile,
    setCurrentUser: noop
});

// eslint-disable-next-line @typescript-eslint/explicit-module-boundary-types
export const useCurrentUserCtx = () => useContext(CurrentUserCtx);
