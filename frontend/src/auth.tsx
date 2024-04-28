import { createContext, ReactNode, useContext } from 'react';
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

const AuthContext = createContext<AuthContext | null>(null);

export function AuthProvider({ children }: { children: ReactNode }) {
    const profileKey = 'profile';

    const login = (profile: UserProfile) => {
        localStorage.setItem(profileKey, JSON.stringify(profile));
    };

    const logout = () => {
        localStorage.removeItem(profileKey);
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
        console.log(`have: ${currentLevel} want: ${wantedLevel}`);
        return currentLevel >= wantedLevel;
    };

    return (
        <AuthContext.Provider
            value={{ profile, logout, isAuthenticated, permissionLevel, hasPermission, login, userSteamID: profile().steam_id }}
        >
            {children}
        </AuthContext.Provider>
    );
}

export const useAuth = (): AuthContext => {
    const context = useContext(AuthContext);
    if (!context) {
        throw new Error('useAuth must be used within an AuthProvider');
    }
    return context;
};

export type AuthContext = {
    profile: () => UserProfile;
    login: (profile: UserProfile) => void;
    logout: () => void;
    isAuthenticated: () => boolean;
    permissionLevel: () => PermissionLevel;
    hasPermission: (level: PermissionLevel) => boolean;
    userSteamID: string;
};
