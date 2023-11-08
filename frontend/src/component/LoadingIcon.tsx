import React from 'react';
import CircularProgress from '@mui/material/CircularProgress';
import { useTheme } from '@mui/material/styles';

export const LoadingIcon = () => {
    const theme = useTheme();
    return (
        <CircularProgress
            sx={{ color: theme.palette.text.primary }}
            size={20}
        />
    );
};
