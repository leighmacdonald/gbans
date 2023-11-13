import React, { useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import Typography from '@mui/material/Typography';
import {
    apiGetCurrentProfile,
    refreshKey,
    tokenKey,
    writeAccessToken,
    writeRefreshToken
} from '../api';
import { GuestProfile, useCurrentUserCtx } from '../contexts/CurrentUserCtx';

const defaultLocation = '/';

export const LoginSteamSuccessPage = () => {
    const navigate = useNavigate();
    const { setCurrentUser } = useCurrentUserCtx();

    useEffect(() => {
        const abortController = new AbortController();

        const loadProfile = async () => {
            const urlParams = new URLSearchParams(window.location.search);
            const refresh = urlParams.get(refreshKey);
            const token = urlParams.get(tokenKey);

            if (!refresh || !token) {
                navigate(defaultLocation);

                return;
            }
            try {
                writeRefreshToken(refresh);
                writeAccessToken(token);

                const profile = await apiGetCurrentProfile(abortController);
                setCurrentUser(profile);
            } catch (e) {
                setCurrentUser(GuestProfile);
            } finally {
                navigate(urlParams.get('next_url') ?? defaultLocation);
            }
        };

        loadProfile();

        return () => abortController.abort();
    });

    return <>{<Typography variant={'h3'}>Logging In...</Typography>}</>;
};
