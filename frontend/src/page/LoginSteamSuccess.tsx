import React, { useEffect } from 'react';
import {
    apiGetCurrentProfile,
    refreshKey,
    tokenKey,
    writeAccessToken,
    writeRefreshToken
} from '../api';
import { useNavigate } from 'react-router-dom';
import Typography from '@mui/material/Typography';
import { GuestProfile, useCurrentUserCtx } from '../contexts/CurrentUserCtx';

const defaultLocation = '/';

export const LoginSteamSuccess = () => {
    const navigate = useNavigate();
    const { setCurrentUser } = useCurrentUserCtx();

    useEffect(() => {
        const urlParams = new URLSearchParams(window.location.search);
        const refresh = urlParams.get(refreshKey);
        const token = urlParams.get(tokenKey);

        if (!refresh || !token) {
            navigate(defaultLocation);

            return;
        }

        writeRefreshToken(refresh);
        writeAccessToken(token);

        apiGetCurrentProfile()
            .then((response) => {
                setCurrentUser(response);
            })
            .catch(() => {
                setCurrentUser(GuestProfile);
            })
            .finally(() => {
                navigate(urlParams.get('next_url') ?? defaultLocation);
            });
    });

    return <>{<Typography variant={'h3'}>Logging In...</Typography>}</>;
};
