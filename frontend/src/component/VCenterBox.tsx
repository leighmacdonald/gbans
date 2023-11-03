import React, { ReactNode } from 'react';
import Box from '@mui/material/Box';

export const VCenterBox = ({ children }: { children: ReactNode }) => (
    <Box sx={{ display: 'flex', alignItems: 'center' }}>{children}</Box>
);
