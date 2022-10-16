import React, { useEffect, useState } from 'react';
import { apiGetCurrentProfile, refreshKey, tokenKey } from '../api';
import { GuestProfile, useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { useNavigate } from 'react-router-dom';
import { logErr } from '../util/errors';
import Typography from '@mui/material/Typography';

const defaultLocation = '/';

export const LoginSuccess = () => {
    //const { sendFlash } = useUserFlashCtx();
    const { setCurrentUser, setToken } = useCurrentUserCtx();
    const navigate = useNavigate();
    const [inProgress, setInProgress] = useState(true);

    useEffect(() => {
        const urlParams = new URLSearchParams(window.location.search);
        const refresh = urlParams.get(refreshKey);
        const token = urlParams.get(tokenKey);
        if (!refresh) {
            logErr('No refresh token received');
            return;
        }
        if (!token) {
            logErr('No auth token received');
            return;
        }
        setToken(token);
        localStorage.setItem(refreshKey, refresh);

        let next_url = urlParams.get('next_url') ?? defaultLocation;

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
                setInProgress(false);
                navigate(next_url);
            });
    }, [navigate, setCurrentUser, setToken]);

    return (
        <>
            {inProgress && (
                <Typography variant={'h3'}>Logging In...</Typography>
            )}
        </>
    );
};
