import { useEffect, JSX } from 'react';
import { Navigate } from 'react-router-dom';
import { createLazyFileRoute } from '@tanstack/react-router';
import { logout } from '../api';
import { useCurrentUserCtx } from '../hooks/useCurrentUserCtx.ts';
import { logErr } from '../util/errors';
import { GuestProfile } from '../util/profile.ts';

export const Route = createLazyFileRoute('/logout')({
    component: LogoutPage
});

function LogoutPage() {
    const { setCurrentUser } = useCurrentUserCtx();

    useEffect(() => {
        const abortController = new AbortController();
        const doLogout = async () => {
            try {
                await logout();
            } catch (e) {
                logErr(e);
            } finally {
                setCurrentUser(GuestProfile);
            }
        };

        doLogout().then(() => {});

        return () => abortController.abort();
    });

    return <Navigate to={'/'} />;
}
