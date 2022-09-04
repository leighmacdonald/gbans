import React, { useEffect } from 'react';
import { apiGetCurrentProfile, tokenKey } from '../api';
import { GuestProfile, useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { useNavigate } from 'react-router-dom';

const defaultLocation = '/';

export const LoginSuccess = () => {
    const { sendFlash } = useUserFlashCtx();
    const { setCurrentUser, setToken } = useCurrentUserCtx();
    const navigate = useNavigate();

    useEffect(() => {
        const urlParams = new URLSearchParams(window.location.search);
        const token = urlParams.get(tokenKey);
        if (!token) {
            return;
        }
        const permissionLevel = urlParams.get('permission_level') || '';
        localStorage.setItem('permission_level', permissionLevel);
        let next_url = urlParams.get('next_url') ?? defaultLocation;
        setToken(token);

        apiGetCurrentProfile()
            .then((response) => {
                // if (!response.status || !response.result) {
                //     sendFlash('error', 'Failed to load profile :(');
                //     return;
                // }
                setCurrentUser(response?.result || GuestProfile);
            })
            .catch(() => {
                next_url = defaultLocation;
            })
            .finally(() => {
                navigate(next_url);
            });
    }, [navigate, sendFlash, setCurrentUser, setToken]);

    return <></>;
};
