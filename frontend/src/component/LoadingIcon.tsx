import React from 'react';
import CircularProgress from '@mui/material/CircularProgress';

export const LoadingIcon = () => {
    return <CircularProgress sx={{ color: 'text.primary' }} size={20} />;
};
