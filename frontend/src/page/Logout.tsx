import React, { useEffect } from 'react';
import { Navigate } from 'react-router-dom';
import { GuestProfile, useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { writeAccessToken, writeRefreshToken } from '../api';

export const Logout = (): JSX.Element => {
    const { setCurrentUser, setToken } = useCurrentUserCtx();

    useEffect(() => {
        setToken('');
        setCurrentUser(GuestProfile);
        writeAccessToken('');
        writeRefreshToken('');
    }, [setCurrentUser, setToken]);

    return <Navigate to={'/'} />;
};
