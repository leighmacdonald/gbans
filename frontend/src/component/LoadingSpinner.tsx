import React from 'react';
import { LoadingButton } from '@mui/lab';
import { useTheme } from '@mui/material/styles';
import CircularProgress from '@mui/material/CircularProgress';

export const LoadingSpinner = () => {
    const theme = useTheme();
    return (
        <LoadingButton
            title={'Loading...'}
            loading
            loadingIndicator={<CircularProgress color="primary" size={24} />}
            variant={'text'}
            color={'secondary'}
            sx={{ color: theme.palette.text.primary }}
        >
            Loading...
        </LoadingButton>
    );
};
