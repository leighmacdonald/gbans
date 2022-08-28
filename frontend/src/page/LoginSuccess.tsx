import React, { useEffect } from 'react';
import { apiGetCurrentProfile, tokenKey } from '../api';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';
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
        const permissionLevel = urlParams.get('permission_level');
        if (permissionLevel) {
            localStorage.setItem('permission_level', permissionLevel);
        }
        const next_url = urlParams.get('next_url') ?? defaultLocation;
        setToken(token);

        apiGetCurrentProfile()
            .then((response) => {
                if (!response.status || !response.result) {
                    sendFlash('error', 'Failed to load profile :(');
                    navigate(defaultLocation);
                    return;
                }
                setCurrentUser(response.result);
                navigate(next_url);
            })
            .catch(() => {
                navigate(defaultLocation);
            });
    }, [navigate, sendFlash, setCurrentUser, setToken]);

    return <></>;
};
