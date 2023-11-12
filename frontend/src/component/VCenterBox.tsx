import React, { ReactNode } from 'react';
import Box from '@mui/material/Box';

export const VCenterBox = ({ children }: { children: ReactNode }) => (
    <Box m={1} display="flex" justifyContent="center" alignItems="center">
        {children}
    </Box>
);
