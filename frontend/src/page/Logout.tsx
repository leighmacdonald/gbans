import React, { useEffect } from 'react';
import { Navigate } from 'react-router-dom';
import { GuestProfile, useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { PermissionLevel } from '../api';

export const Logout = (): JSX.Element => {
    const { setCurrentUser, setToken } = useCurrentUserCtx();

    useEffect(() => {
        setToken('');
        setCurrentUser(GuestProfile);
        localStorage.setItem('permission_level', `${PermissionLevel.Guest}`);
        localStorage.removeItem('token');
    }, [setCurrentUser, setToken]);

    return <Navigate to={'/'} />;
};
