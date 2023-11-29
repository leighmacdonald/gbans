import React, { ReactNode } from 'react';
import Box from '@mui/material/Box';

export const VCenterBox = ({
    children,
    justify = 'center'
}: {
    children: ReactNode;
    justify?: 'left' | 'center';
}) => (
    <Box
        m={1}
        display="flex"
        justifyContent={justify}
        alignItems="center"
        margin={0}
    >
        {children}
    </Box>
);
