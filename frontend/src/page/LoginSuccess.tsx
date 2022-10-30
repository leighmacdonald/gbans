import React, { useEffect, useState } from 'react';
import {
    apiGetCurrentProfile,
    refreshKey,
    tokenKey,
    writeAccessToken,
    writeRefreshToken
} from '../api';
import { GuestProfile, useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { useNavigate } from 'react-router-dom';
import { logErr } from '../util/errors';
import Typography from '@mui/material/Typography';

const defaultLocation = '/';

export const LoginSuccess = () => {
    //const { sendFlash } = useUserFlashCtx();
    const { setCurrentUser } = useCurrentUserCtx();
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
        writeRefreshToken(refresh);
        writeAccessToken(token);

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
    }, [navigate, setCurrentUser]);

    return (
        <>
            {inProgress && (
                <Typography variant={'h3'}>Logging In...</Typography>
            )}
        </>
    );
};
