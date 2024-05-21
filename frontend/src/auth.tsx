import { createContext, ReactNode } from 'react';
import { PermissionLevel, UserProfile } from './api';
import { guestProfile } from './util/auth/guestProfile.ts';
import { logoutFn } from './util/auth/logoutFn.ts';

export const refreshKey = 'refresh';
export const tokenKey = 'token';
export const profileKey = 'profile';
export const logoutKey = 'logout';

export const AuthContext = createContext<AuthContext | null>(null);

export function AuthProvider({
    children,
    profile,
    setProfile
}: {
    children: ReactNode;
    profile: UserProfile;
    setProfile: (v?: UserProfile) => void;
}) {
    const login = (profile: UserProfile) => {
        localStorage.setItem(profileKey, JSON.stringify(profile));
        setProfile(profile);
    };

    const logout = async () => {
        try {
            await logoutFn();
        } catch (e) {
            console.log(`error logging out: ${e}`);
        } finally {
            setProfile(guestProfile);
        }
    };

    const isAuthenticated = () => {
        return Boolean(profile?.steam_id ?? false);
    };

    const permissionLevel = () => {
        return profile?.permission_level ?? PermissionLevel.Guest;
    };

    const hasPermission = (wantedLevel: PermissionLevel) => {
        const currentLevel = permissionLevel();
        return currentLevel >= wantedLevel;
    };

    return (
        <AuthContext.Provider
            value={{
                profile,
                logout,
                isAuthenticated,
                permissionLevel,
                hasPermission,
                login
            }}
        >
            {children}
        </AuthContext.Provider>
    );
}

export type AuthContext = {
    profile: UserProfile;
    login: (profile: UserProfile) => void;
    logout: () => Promise<void>;
    isAuthenticated: () => boolean;
    permissionLevel: () => PermissionLevel;
    hasPermission: (level: PermissionLevel) => boolean;
};
