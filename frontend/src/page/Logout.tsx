import React from 'react';
import { Redirect } from 'react-router';
import { handleOnLogout } from '../util/api';

export const Logout = (): JSX.Element => {
    handleOnLogout();
    return <Redirect to={'/'} />;
};
