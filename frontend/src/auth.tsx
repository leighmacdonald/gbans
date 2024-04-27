import { useState } from 'react';
import { defaultAvatarHash, PermissionLevel, UserProfile } from './api';

const guestProfile: UserProfile = {
    steam_id: '',
    permission_level: PermissionLevel.Guest,
    avatarhash: defaultAvatarHash,
    name: '',
    ban_id: 0,
    muted: false,
    discord_id: '',
    created_on: new Date(),
    updated_on: new Date()
};
export const useAuth = (): AuthContext => {
    const [user, setUser] = useState<UserProfile>(guestProfile);

    const login = (profile: UserProfile) => {
        setUser(profile);
    };

    const logout = () => {
        setUser(guestProfile);
    };

    const isAuthenticated = () => {
        return user.steam_id != '';
    };

    const permissionLevel = () => {
        return user ? user.permission_level : PermissionLevel.Guest;
    };

    const hasPermission = (wantedLevel: PermissionLevel) => {
        return permissionLevel() >= wantedLevel;
    };

    return { login, user, logout, isAuthenticated, permissionLevel, userSteamID: user ? user.steam_id : '', hasPermission };
};

export type AuthContext = {
    user: UserProfile;
    login: (profile: UserProfile) => void;
    logout: () => void;
    isAuthenticated: () => boolean;
    permissionLevel: () => PermissionLevel;
    hasPermission: (level: PermissionLevel) => boolean;
    userSteamID: string;
};
