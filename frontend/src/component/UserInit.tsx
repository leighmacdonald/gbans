import React from 'react';
import { useEffect } from 'react';
import {
    apiGetCurrentProfile,
    readAccessToken,
    readRefreshToken,
    refreshToken
} from '../api';
import { GuestProfile, useCurrentUserCtx } from '../contexts/CurrentUserCtx';
import { emptyOrNullString } from '../util/types';

export const UserInit = () => {
    const { setCurrentUser, currentUser } = useCurrentUserCtx();

    useEffect(() => {
        if (currentUser.steam_id != GuestProfile.steam_id) {
            // Don't bother re-loading if we are already did in from the login success page
            return;
        }
        const at = readAccessToken();
        const rt = readRefreshToken();
        if (!emptyOrNullString(at) && !emptyOrNullString(rt)) {
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
        } else if (!emptyOrNullString(rt)) {
            refreshToken().then(() => {
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
            });
        }

        // eslint-disable-next-line
    }, []);

    return <></>;
};
