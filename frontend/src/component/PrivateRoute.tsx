import React from 'react';
import { Navigate, useLocation } from 'react-router-dom';
import { RouteProps } from 'react-router';

export interface PrivateRouteProps extends RouteProps {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    children: JSX.Element;
    permission: number;
}

export const PrivateRoute = ({ children, permission }: PrivateRouteProps) => {
    const permission_level = parseInt(
        localStorage.getItem('permission_level') || '1'
    );
    const location = useLocation();
    if (permission_level >= permission) {
        // Redirect them to the /login page, but save the current location they were
        // trying to go to when they were redirected. This allows us to send them
        // along to that page after they login, which is a nicer user experience
        // than dropping them off on the home page.
        return <Navigate to="/login" state={{ from: location }} />;
    }

    return children;
};
