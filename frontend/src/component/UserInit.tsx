import { useEffect } from 'react';
import { apiGetCurrentProfile, readRefreshToken } from '../api';
import { useCurrentUserCtx } from '../hooks/useCurrentUserCtx.ts';
import { GuestProfile } from '../util/profile.ts';
import { emptyOrNullString } from '../util/types';

export const UserInit = () => {
    const { setCurrentUser, currentUser } = useCurrentUserCtx();

    useEffect(() => {
        if (currentUser.steam_id != GuestProfile.steam_id) {
            // Don't bother re-loading if we are already did in from the login success page
            return;
        }
        const abortController = new AbortController();

        const loadProfile = async () => {
            try {
                const rt = readRefreshToken();
                if (!emptyOrNullString(rt)) {
                    setCurrentUser(await apiGetCurrentProfile(abortController));
                } else {
                    setCurrentUser(GuestProfile);
                }
            } catch (e) {
                setCurrentUser(GuestProfile);
            }
        };

        loadProfile();

        return () => abortController.abort();
    });

    return <></>;
};
