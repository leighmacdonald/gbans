import { useEffect, useState } from 'react';
import Typography from '@mui/material/Typography';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { apiLinkDiscord } from '../api';
import { logErr } from '../util/errors.ts';

export const Route = createFileRoute('/_auth/login/discord')({
    component: LoginDiscordSuccess
});

function LoginDiscordSuccess() {
    const navigate = useNavigate();
    const [inProgress, setInProgress] = useState(true);
    const next_url = '/settings';

    useEffect(() => {
        const urlParams = new URLSearchParams(window.location.search);
        const code = urlParams.get('code');
        if (!code) {
            navigate({ to: next_url });
            return;
        }
        apiLinkDiscord({ code })
            .catch((e) => {
                logErr(e);
            })
            .finally(() => {
                setInProgress(false);
                navigate({ to: next_url });
            });
    }, [navigate]);

    return <>{inProgress && <Typography variant={'h3'}>Logging In...</Typography>}</>;
}
