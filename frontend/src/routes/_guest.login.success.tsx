import { useEffect } from 'react';
import Typography from '@mui/material/Typography';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute, useLayoutEffect, useRouter } from '@tanstack/react-router';
import { z } from 'zod';
import { apiGetCurrentProfile } from '../api';
import { writeAccessToken, writeRefreshToken } from '../auth.tsx';

export const Route = createFileRoute('/_guest/login/success')({
    validateSearch: z.object({
        next_url: z.string().optional().catch(''),
        refresh: z.string(),
        token: z.string()
    })
}).update({
    component: LoginSteamSuccess
});

function LoginSteamSuccess() {
    const router = useRouter();
    const { login } = Route.useRouteContext({
        select: ({ auth }) => auth
    });
    const search = Route.useSearch();
    const { data: profile } = useQuery({
        queryKey: ['user'],
        queryFn: apiGetCurrentProfile
    });

    writeRefreshToken(search.refresh);
    writeAccessToken(search.token);

    useEffect(() => {
        if (!profile) {
            return;
        }

        login(profile);
        router.invalidate();
    }, [login, profile, router, search.refresh, search.token]);

    useLayoutEffect(() => {
        if (!profile) {
            return;
        }

        if (profile.steam_id != '' && search.next_url) {
            router.history.push(search.next_url);
        }
    }, [profile, router.history, search, search.next_url]);

    return <>{<Typography variant={'h3'}>Logging In...</Typography>}</>;
}
