import React from 'react';
import { Navigate } from 'react-router';
import { apiGetCurrentProfile } from '../api';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';

export const LoginSuccess = (): JSX.Element => {
    const { sendFlash } = useUserFlashCtx();
    const { setCurrentUser } = useCurrentUserCtx();
    const urlParams = new URLSearchParams(window.location.search);
    const token = urlParams.get('token');
    const perms = urlParams.get('permission_level');
    if (token != null && token.length > 0) {
        localStorage.setItem('token', token);
        localStorage.setItem('permission_level', `${perms}`);
    }
    let next_url = urlParams.get('next_url');
    if (next_url == null || next_url == '') {
        next_url = '/';
    }

    apiGetCurrentProfile().then((response) => {
        if (!response.status || !response.result) {
            sendFlash('error', 'Failed to load profile');
            alert('bye');
            return;
        }
        setCurrentUser(response.result);
    });
    return <Navigate to={next_url} />;
};
