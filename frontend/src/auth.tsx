import { createContext, ReactNode, useCallback, useEffect } from 'react';
import { apiGetCurrentProfile, defaultAvatarHash, PermissionLevel, UserProfile } from './api';
import { logoutFn } from './util/auth/logoutFn.ts';
import { readAccessToken } from './util/auth/readAccessToken.ts';
import { emptyOrNullString } from './util/types.ts';

export const accessTokenKey = 'token';
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
    const login = useCallback(
        async (profile: UserProfile) => {
            localStorage.setItem(profileKey, JSON.stringify(profile));
            setProfile(profile);
        },
        [setProfile]
    );

    const logout = useCallback(async () => {
        try {
            await logoutFn();
        } catch (e) {
            console.log(`error logging out: ${e}`);
        } finally {
            setProfile({
                steam_id: '',
                permission_level: PermissionLevel.Guest,
                avatarhash: defaultAvatarHash,
                name: '',
                ban_id: 0,
                muted: false,
                discord_id: '',
                created_on: new Date(),
                updated_on: new Date(),
                patreon_id: ''
            });
        }
    }, [setProfile]);

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

    useEffect(() => {
        const loadProfile = async () => {
            try {
                const token = readAccessToken();
                if (!emptyOrNullString(token)) {
                    await login(await apiGetCurrentProfile());
                }
            } catch (e) {
                await logout();
            }
        };
        loadProfile();
    }, [login, logout]);

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
