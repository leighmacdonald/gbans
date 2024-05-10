import { createContext, ReactNode } from 'react';
import { defaultAvatarHash, PermissionLevel, UserProfile } from './api';
import { logoutFn } from './util/auth/logoutFn.ts';

export const refreshKey = 'refresh';
export const tokenKey = 'token';
export const profileKey = 'profile';
export const logoutKey = 'logout';

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

export const AuthContext = createContext<AuthContext | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
    const login = (profile: UserProfile) => {
        localStorage.setItem(profileKey, JSON.stringify(profile));
    };

    const profile = (): UserProfile => {
        try {
            const userData = localStorage.getItem(profileKey);
            if (!userData) {
                return guestProfile;
            }

            return JSON.parse(userData);
        } catch (e) {
            return guestProfile;
        }
    };

    const isAuthenticated = () => {
        return profile().steam_id != '';
    };

    const permissionLevel = () => {
        return profile().permission_level;
    };

    const hasPermission = (wantedLevel: PermissionLevel) => {
        const currentLevel = permissionLevel();
        return currentLevel >= wantedLevel;
    };

    return (
        <AuthContext.Provider
            value={{
                profile,
                logout: logoutFn,
                isAuthenticated,
                permissionLevel,
                hasPermission,
                login,
                userSteamID: profile().steam_id
            }}
        >
            {children}
        </AuthContext.Provider>
    );
}

export type AuthContext = {
    profile: () => UserProfile;
    login: (profile: UserProfile) => void;
    logout: () => Promise<void>;
    isAuthenticated: () => boolean;
    permissionLevel: () => PermissionLevel;
    hasPermission: (level: PermissionLevel) => boolean;
    userSteamID: string;
};
