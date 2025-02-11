import Typography from '@mui/material/Typography';
import { ApiError } from '../error.tsx';

export const ErrorBar = ({ error }: { error: ApiError }) => {
    return (
        <div style={{ backgroundColor: 'pink', border: '2px solid red', width: '100%' }}>
            <Typography textAlign="center" variant="body2" color={'error'} padding={1} fontWeight={700}>
                {error.code}: {error.message}
            </Typography>
        </div>
    );
};
