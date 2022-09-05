import React from 'react';
import { RouteProps } from 'react-router';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { PermissionLevel } from '../api';
import { Navigate } from 'react-router-dom';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';

export interface PrivateRouteProps extends RouteProps {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    children: JSX.Element;
    permission: number;
}

export const PrivateRoute = ({ children, permission }: PrivateRouteProps) => {
    const { currentUser, token } = useCurrentUserCtx();
    const { sendFlash } = useUserFlashCtx();

    if (!token) {
        // No token means we have no login credentials stored at all, so login first.
        return <Navigate to={'/login'} />;
    }
    const storedLevel = parseInt(
        localStorage.getItem('permission_level') || `${PermissionLevel.Guest}`
    );
    // This is to handle the race for loading the user profile without redirecting to the homepage.
    // If the stored permission_level is enough, allow the user to load the page without redirect, it
    // will be checked against the actual currentUser once its changed/loaded
    if (
        !currentUser.steam_id.isValidIndividual() &&
        storedLevel >= permission
    ) {
        return children;
    }
    if (currentUser && currentUser.permission_level < permission) {
        // Redirect them to the /login page, but save the current location they were
        // trying to go to when they were redirected. This allows us to send them
        // along to that page after they login, which is a nicer user experience
        // than dropping them off on the home page.
        sendFlash('error', 'Permission denied');
        return <Navigate to={'/'} />;
    }
    return children;
};
