import { useMemo, JSX } from 'react';
import { RouteProps } from 'react-router';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { LoginPage } from '../page/LoginPage';

export interface PrivateRouteProps {
    children: JSX.Element;
    permission: number;
}

export const PrivateRoute = ({
    children,
    permission
}: PrivateRouteProps & RouteProps): JSX.Element => {
    const { currentUser } = useCurrentUserCtx();
    const canView = useMemo(() => {
        return currentUser && currentUser.permission_level >= permission;
    }, [currentUser, permission]);
    return canView ? children : <LoginPage />;
};
