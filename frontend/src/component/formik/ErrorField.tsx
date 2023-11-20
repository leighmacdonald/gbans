import React from 'react';
import Box from '@mui/material/Box';
import Typography from '@mui/material/Typography';
import { useTheme } from '@mui/material/styles';
import { emptyOrNullString } from '../../util/types';

export const ErrorField = ({ error }: { error?: string }) => {
    const theme = useTheme();

    if (emptyOrNullString(error)) {
        return <></>;
    }
    return (
        <Box>
            <Typography variant={'subtitle1'} color={theme.palette.error.main}>
                Error: {error}
            </Typography>
        </Box>
    );
};
