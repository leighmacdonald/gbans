import React, { ReactNode } from 'react';
import { LoadingIcon } from './LoadingIcon';

export const LoadingHeaderIcon = ({
    loading,
    icon
}: {
    loading: boolean;
    icon: ReactNode;
}) => {
    return loading ? <LoadingIcon /> : icon;
};
