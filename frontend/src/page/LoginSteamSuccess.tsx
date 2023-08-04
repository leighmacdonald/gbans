import React, { useEffect } from 'react';
import { refreshKey, tokenKey } from '../api';
import { useNavigate } from 'react-router-dom';
import Typography from '@mui/material/Typography';
import { useCurrentUserCtx } from '../contexts/CurrentUserCtx';

const defaultLocation = '/';

export const LoginSteamSuccess = () => {
    const { setRefreshToken, setToken } = useCurrentUserCtx();
    const navigate = useNavigate();

    useEffect(() => {
        const urlParams = new URLSearchParams(window.location.search);
        const refresh = urlParams.get(refreshKey);
        const token = urlParams.get(tokenKey);
        if (!refresh || !token) {
            return;
        }
        setRefreshToken(refresh);
        setToken(token);

        navigate(urlParams.get('next_url') ?? defaultLocation);
    });

    return <>{<Typography variant={'h3'}>Logging In...</Typography>}</>;
};
