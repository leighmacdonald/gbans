import React from 'react';
import { RouteProps, useLocation } from 'react-router';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { Navigate } from 'react-router-dom';

export interface PrivateRouteProps extends RouteProps {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    children: JSX.Element;
    permission: number;
}

export const PrivateRoute = ({ children, permission }: PrivateRouteProps) => {
    const { currentUser } = useCurrentUserCtx();
    const location = useLocation();
    if (!currentUser || currentUser.permission_level < permission) {
        // Redirect them to the /login page, but save the current location they were
        // trying to go to when they were redirected. This allows us to send them
        // along to that page after they login, which is a nicer user experience
        // than dropping them off on the home page.
        return <Navigate to="/login" state={{ from: location }} />;
    }
    return children;
};
