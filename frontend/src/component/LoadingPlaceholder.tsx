import Box from '@mui/material/Box';
import { LoadingSpinner } from './LoadingSpinner';

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
