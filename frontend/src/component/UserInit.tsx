import React from 'react';
import { useEffect } from 'react';
import { apiGetCurrentProfile, readRefreshToken } from '../api';
import { GuestProfile, useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { emptyOrNullString } from '../util/types';

export const UserInit = () => {
    const { setCurrentUser, currentUser, setRefreshToken } =
        useCurrentUserCtx();

    useEffect(() => {
        if (currentUser.steam_id != GuestProfile.steam_id) {
            // Don't bother re-loading if we are already did in from the login success page
            return;
        }
        const rt = readRefreshToken();
        if (!emptyOrNullString(rt)) {
            apiGetCurrentProfile()
                .then((response) => {
                    if (!response.status || !response.result) {
                        return;
                    }
                    setCurrentUser(response.result);
                })
                .catch(() => {
                    setCurrentUser(GuestProfile);
                });
        }

        // eslint-disable-next-line
    }, [setRefreshToken]);

    return <></>;
};
