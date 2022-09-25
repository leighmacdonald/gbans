import React from 'react';
import { RouteProps } from 'react-router';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { Login } from '../page/Login';

export interface PrivateRouteProps extends RouteProps {
    children: JSX.Element;
    permission: number;
}

export const PrivateRoute = ({
    children,
    permission
}: PrivateRouteProps): JSX.Element => {
    const { currentUser, token } = useCurrentUserCtx();
    if (token && currentUser && currentUser.permission_level >= permission) {
        return children;
    }
    return <Login />;
};
