import { ReactNode, useContext, useState } from 'react';
import { PermissionLevel, UserProfile } from '../api';
import { CurrentUserCtx } from '../contexts/CurrentUserCtx.tsx';
import { GuestProfile } from '../util/profile.ts';

export function useCurrentUserCtx() {
    const context = useContext(CurrentUserCtx);
    if (!context) {
        throw new Error('useCurrentUserCtx must be used within AuthProvider');
    }

    return context;
}

export function AuthProvider({ children }: { children: ReactNode }) {
    const [currentUser, setCurrentUser] =
        useState<NonNullable<UserProfile>>(GuestProfile);

    const isAuthenticated =
        currentUser.permission_level != PermissionLevel.Guest;
    const permissionLevel = currentUser.permission_level;

    return (
        <CurrentUserCtx.Provider
            value={{
                currentUser,
                setCurrentUser,
                isAuthenticated,
                permissionLevel
            }}
        >
            {children}
        </CurrentUserCtx.Provider>
    );
}
