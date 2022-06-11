import React from 'react';
import { Navigate } from 'react-router-dom';
import { handleOnLogout } from '../api';

export const Logout = (): JSX.Element => {
    handleOnLogout();
    return <Navigate to={'/'} />;
};
