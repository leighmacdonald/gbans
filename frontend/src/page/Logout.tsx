import React, { useEffect, JSX } from 'react';
import { Navigate } from 'react-router-dom';
import { GuestProfile, useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { logout } from '../api';

export const Logout = (): JSX.Element => {
    const { setCurrentUser } = useCurrentUserCtx();

    useEffect(() => {
        logout();
        setCurrentUser(GuestProfile);
    }, [setCurrentUser]);

    return <Navigate to={'/'} />;
};
