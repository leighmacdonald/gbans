import { ReactNode, useCallback, useEffect } from 'react';
import { apiGetCurrentProfile, defaultAvatarHash } from './api';
import { AuthContext } from './contexts/AuthContext.tsx';
import { PermissionLevel, PermissionLevelEnum, UserProfile } from './schema/people.ts';
import { logoutFn } from './util/auth/logoutFn.ts';
import { readAccessToken } from './util/auth/readAccessToken.ts';
import { logErr } from './util/errors.ts';
import { emptyOrNullString } from './util/types.ts';

export const accessTokenKey = 'token';
export const profileKey = 'profile';
export const logoutKey = 'logout';

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
            logErr(`error logging out: ${e}`);
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
                patreon_id: '',
                playerqueue_chat_status: 'noaccess',
                playerqueue_chat_reason: ''
            });
        }
    }, [setProfile]);

    const isAuthenticated = () => {
        return Boolean(profile?.steam_id ?? false);
    };

    const permissionLevel = () => {
        return profile?.permission_level ?? PermissionLevel.Guest;
    };

    const hasPermission = (wantedLevel: PermissionLevelEnum) => {
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
                logErr(e);
                await logout();
            }
        };
        loadProfile().catch(logErr);
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

export type AuthContextProps = {
    profile: UserProfile;
    login: (profile: UserProfile) => void;
    logout: () => Promise<void>;
    isAuthenticated: () => boolean;
    permissionLevel: () => PermissionLevelEnum;
    hasPermission: (level: PermissionLevelEnum) => boolean;
};
