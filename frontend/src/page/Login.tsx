import React, { useEffect } from 'react';
import { handleOnLogin } from '../api';

export const Login = (): JSX.Element => {
    useEffect(() => {
        handleOnLogin(window.location.pathname);
    }, []);
    return <></>;
};
