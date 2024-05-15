import Box from '@mui/material/Box';
import { LoadingSpinner } from './LoadingSpinner';

export const LoadingPlaceholder = ({ height = 400 }: { height?: number }) => {
    return (
        <Box height={height} display="flex" justifyContent="center" alignItems="center">
            <LoadingSpinner />
        </Box>
    );
};
