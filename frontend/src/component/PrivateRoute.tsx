import React from 'react';
import { Route } from 'react-router-dom';
import { RouteProps } from 'react-router';

export interface PrivateRouteProps extends RouteProps {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    component: any;
    permission: number;
}

export const PrivateRoute = ({
    component: Component,
    permission: permission,
    ...rest
}: PrivateRouteProps): JSX.Element => {
    const permission_level = parseInt(
        localStorage.getItem('permission_level') || '1'
    );
    if (permission_level >= permission) {
        return <></>;
    }
    return <Route {...rest} render={(props) => <Component {...props} />} />;
};
