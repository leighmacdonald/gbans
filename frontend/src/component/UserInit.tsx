import React from 'react';
import { useEffect } from 'react';
import { apiGetCurrentProfile } from '../api';
import { GuestProfile, useCurrentUserCtx } from '../contexts/CurrentUserCtx';

export const UserInit = () => {
    const { setCurrentUser } = useCurrentUserCtx();

    useEffect(() => {
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
        // eslint-disable-next-line
    }, []);

    return <></>;
};
