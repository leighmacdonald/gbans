import {createContext, useContext} from 'react';
import {communityVisibilityState, PlayerProfile} from '../util/api';

export interface StoreProps {
    children: JSX.Element[];
}

// export const AuthStore = ({children}: StoreProps) => {
//     const [state] = useReducer<React.Reducer<PlayerProfile, Action>>(
//         Reducer, initialState)
//     return (
//         <CurrentUserCtx.Provider value={state} children={children} />
//     )
// }

type ReducerActionType = 'SET_PROFILE' | 'SET_ERROR';

export interface Action {
    type: ReducerActionType;
    payload: any;
}

export const Reducer = (state: PlayerProfile, action: Action): PlayerProfile => {
    switch (action.type) {
        case 'SET_PROFILE':
            return action.payload;
        case 'SET_ERROR':
            return action.payload;
        default:
            return state;
    }
};

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
        steam_id: 0,
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
    setCurrentUser: _ => {}
});

export const useCurrentUserCtx = () => useContext(CurrentUserCtx);
