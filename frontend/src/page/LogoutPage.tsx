import { useEffect, JSX } from 'react';
import { Navigate } from 'react-router-dom';
import { logout } from '../api';
import { GuestProfile, useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { logErr } from '../util/errors';

export const LogoutPage = (): JSX.Element => {
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
};

export default LogoutPage;
