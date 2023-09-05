import { LoadingSpinner } from './LoadingSpinner';
import Box from '@mui/material/Box';
import React from 'react';

export const LoadingPlaceholder = () => {
    return (
        <Box
            height={400}
            display="flex"
            justifyContent="center"
            alignItems="center"
        >
            <LoadingSpinner />
        </Box>
    );
};
