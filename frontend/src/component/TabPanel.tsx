import Box from '@mui/material/Box';
import React from 'react';

export interface TabPanelProps {
    children?: React.ReactNode;
    index: number;
    value: number;
}

export const TabPanel = (props: TabPanelProps) => {
    const { children, value, index, ...other } = props;

    return (
        <div
            role="tabpanel"
            hidden={value !== index}
            id={`tabpanel-${index}`}
            aria-labelledby={`ab-${index}`}
            {...other}
        >
            {value === index && <Box sx={{ p: 0 }}>{children}</Box>}
        </div>
    );
};
