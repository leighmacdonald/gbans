import React from 'react';
import { logout, logoutKey } from '../api';

export const LogoutHandler = () => {
    // Listen for storage events with the logout key and logout from all browser sessions/tabs when fired.
    window.addEventListener('storage', (event) => {
        if (event.key === logoutKey) {
            logout();
            document.location.reload();
        }
    });

    return <></>;
};
