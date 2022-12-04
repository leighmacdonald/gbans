import React, { useEffect, useState } from 'react';
import { apiLinkDiscord } from '../api';
import { useNavigate } from 'react-router-dom';
import Typography from '@mui/material/Typography';
import { logErr } from '../util/errors';

export const LoginDiscordSuccess = () => {
    const navigate = useNavigate();
    const [inProgress, setInProgress] = useState(true);
    const next_url = '/settings';

    useEffect(() => {
        const urlParams = new URLSearchParams(window.location.search);
        const code = urlParams.get('code');
        if (!code) {
            navigate(next_url);
            return;
        }
        apiLinkDiscord({ code })
            .then((response) => {
                if (!response.status) {
                    return;
                }
            })
            .catch((e) => {
                logErr(e);
            })
            .finally(() => {
                setInProgress(false);
                navigate(next_url);
            });
    }, []);

    return (
        <>
            {inProgress && (
                <Typography variant={'h3'}>Logging In...</Typography>
            )}
        </>
    );
};
