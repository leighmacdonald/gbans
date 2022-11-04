import React, { useEffect, useState } from 'react';
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

export const LoginSuccess = () => {
    const { setCurrentUser } = useCurrentUserCtx();
    const navigate = useNavigate();
    const [inProgress, setInProgress] = useState(true);

    useEffect(() => {
        const urlParams = new URLSearchParams(window.location.search);
        const refresh = urlParams.get(refreshKey);
        const token = urlParams.get(tokenKey);
        if (!refresh || !token) {
            return;
        }
        writeRefreshToken(refresh);
        writeAccessToken(token);

        const next_url = urlParams.get('next_url') ?? defaultLocation;
        localStorage.removeItem('token'); // cleanup old key
        apiGetCurrentProfile()
            .then((response) => {
                if (!response.status || !response.result) {
                    return;
                }
                setCurrentUser(response.result);
            })
            .catch(() => {
                setCurrentUser(GuestProfile);
            })
            .finally(() => {
                setInProgress(false);
                navigate(next_url);
            });

        // apiGetCurrentProfile()
        //     .then((response) => {
        //         // if (!response.status || !response.result) {
        //         //     sendFlash('error', 'Failed to load profile :(');
        //         //     return;
        //         // }
        //         setCurrentUser(response?.result || GuestProfile);
        //     })
        //     .catch(() => {
        //         next_url = defaultLocation;
        //     })
        //     .finally(() => {
        //         setInProgress(false);
        //         navigate(next_url);
        //     });
        // eslint-disable-next-line
    }, []);

    return (
        <>
            {inProgress && (
                <Typography variant={'h3'}>Logging In...</Typography>
            )}
        </>
    );
};
