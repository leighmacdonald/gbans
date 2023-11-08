import React from 'react';
import { LoadingButton } from '@mui/lab';
import { useTheme } from '@mui/material/styles';
import { LoadingIcon } from './LoadingIcon';

export const LoadingSpinner = () => {
    const theme = useTheme();
    return (
        <LoadingButton
            title={'Loading...'}
            loading
            loadingIndicator={<LoadingIcon />}
            variant={'text'}
            color={'secondary'}
            sx={{ color: theme.palette.text.primary }}
        >
            Loading...
        </LoadingButton>
    );
};
