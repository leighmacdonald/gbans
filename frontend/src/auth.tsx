import { createContext, ReactNode, useContext } from 'react';
import { apiCall, defaultAvatarHash, EmptyBody, PermissionLevel, UserProfile } from './api';

export const refreshKey = 'refresh';
export const tokenKey = 'token';
export const profileKey = 'profile';
export const logoutKey = 'logout';

export const writeAccessToken = (token: string) => {
    if (token == '') {
        sessionStorage.removeItem(tokenKey);
    } else {
        sessionStorage.setItem(tokenKey, token);
    }
};

export const readAccessToken = () => {
    try {
        return sessionStorage.getItem(tokenKey) as string;
    } catch (e) {
        return '';
    }
};

export const writeRefreshToken = (token: string) => {
    if (token == '') {
        localStorage.removeItem(refreshKey);
    } else {
        localStorage.setItem(refreshKey, token);
    }
};

export const readRefreshToken = () => {
    try {
        return localStorage.getItem(refreshKey) as string;
    } catch (e) {
        return '';
    }
};

const baseUrl = () => {
    let returnUrl = window.location.hostname;
    if (window.location.port !== '') {
        returnUrl = `${returnUrl}:${window.location.port}`;
    }
    return `${window.location.protocol}//${returnUrl}`;
};

export const generateOIDCLink = (returnPath: string): string => {
    let returnUrl = window.location.hostname;
    if (window.location.port !== '') {
        returnUrl = `${returnUrl}:${window.location.port}`;
    }
    // Don't redirect loop to /login
    const returnTo = `${window.location.protocol}//${returnUrl}/auth/callback?return_url=${returnPath !== '/login' ? returnPath : '/'}`;

    return [
        'https://steamcommunity.com/openid/login',
        `?openid.ns=${encodeURIComponent('http://specs.openid.net/auth/2.0')}`,
        '&openid.mode=checkid_setup',
        `&openid.return_to=${encodeURIComponent(returnTo)}`,
        `&openid.realm=${encodeURIComponent(`${window.location.protocol}//${window.location.hostname}`)}`,
        `&openid.ns.sreg=${encodeURIComponent('http://openid.net/extensions/sreg/1.1')}`,
        `&openid.claimed_id=${encodeURIComponent('http://specs.openid.net/auth/2.0/identifier_select')}`,
        `&openid.identity=${encodeURIComponent('http://specs.openid.net/auth/2.0/identifier_select')}`
    ].join('');
};

export const discordLoginURL = () => {
    return (
        'https://discord.com/oauth2/authorize' +
        '?client_id=' +
        window.gbans.discord_client_id +
        '&redirect_uri=' +
        encodeURIComponent(baseUrl() + '/login/discord') +
        '&response_type=code' +
        '&scope=identify'
    );
};

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

export const logoutFn = async () => {
    localStorage.removeItem(profileKey);
    localStorage.removeItem(tokenKey);
    localStorage.removeItem(refreshKey);
    localStorage.setItem(logoutKey, Date.now().toString());
    await apiCall<EmptyBody>('/api/auth/logout', 'GET', undefined);
};

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
            value={{ profile, logout: logoutFn, isAuthenticated, permissionLevel, hasPermission, login, userSteamID: profile().steam_id }}
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
    logout: () => Promise<void>;
    isAuthenticated: () => boolean;
    permissionLevel: () => PermissionLevel;
    hasPermission: (level: PermissionLevel) => boolean;
    userSteamID: string;
};
