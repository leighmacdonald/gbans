import { createContext, useContext } from 'react';
import { communityVisibilityState, PlayerProfile } from '../api';
import { noop } from 'lodash-es';

export const GuestProfile: PlayerProfile = {
    player: {
        personaname: 'Guest',
        avatar: '',
        avatarfull: '',
        avatarhash: '',
        avatarmedium: '',
        communityvisibilitystate: communityVisibilityState.Private,
        created_on: new Date(),
        ip_addr: '',
        loccityid: 0,
        loccountrycode: '',
        locstatecode: '',
        personastate: 0,
        personastateflags: 0,
        primaryclanid: '',
        profilestate: 0,
        profileurl: '',
        realname: '',
        steam_id: '',
        steamid: '',
        timecreated: 0,
        updated_on: new Date()
    },
    friends: []
};

export type CurrentUser = {
    currentUser: PlayerProfile;
    setCurrentUser: (profile: PlayerProfile) => void;
};

export const CurrentUserCtx = createContext<CurrentUser>({
    currentUser: GuestProfile,
    setCurrentUser: noop
});

// eslint-disable-next-line @typescript-eslint/explicit-module-boundary-types
export const useCurrentUserCtx = () => useContext(CurrentUserCtx);
