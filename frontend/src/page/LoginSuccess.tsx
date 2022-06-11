import React from 'react';
import { Navigate } from 'react-router';
import { apiGetCurrentProfile, PermissionLevel, UserProfile } from '../api';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';

export const LoginSuccess = (): JSX.Element => {
    const { setCurrentUser } = useCurrentUserCtx();
    const urlParams = new URLSearchParams(window.location.search);
    const token = urlParams.get('token');
    if (token != null && token.length > 0) {
        localStorage.setItem('token', token);
        localStorage.setItem(
            'permission_level',
            `${
                urlParams.get('permission_level') ??
                PermissionLevel.Authenticated
            }`
        );
    }
    let next_url = urlParams.get('next_url');
    if (next_url == null || next_url == '') {
        next_url = '/';
    }

    apiGetCurrentProfile().then((value) => {
        setCurrentUser(value as UserProfile);
    });
    const { flashes, setFlashes } = useUserFlashCtx();
    setFlashes([
        ...flashes,
        {
            closable: true,
            heading: 'header',
            level: 'success',
            message: 'Login Successful'
        }
    ]);
    return <Navigate to={next_url} />;
};
