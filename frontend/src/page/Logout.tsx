import React from 'react';
import { Navigate } from 'react-router-dom';
import { handleOnLogout } from '../api/auth';

export const Logout = (): JSX.Element => {
    handleOnLogout();
    return <Navigate to={'/'} />;
};
