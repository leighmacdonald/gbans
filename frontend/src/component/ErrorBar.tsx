import Typography from '@mui/material/Typography';
import { AppError } from '../error.tsx';

export const ErrorBar = ({ error }: { error: AppError }) => {
    return (
        <div style={{ backgroundColor: 'pink', border: '2px solid red', width: '100%' }}>
            <Typography
                padding={1}
                textAlign="center"
                variant="body2"
                color={'error'}
                fontWeight={700}
                sx={{ textTransform: 'capitalize' }}
            >
                {error.name}
            </Typography>
            <Typography textAlign="center" variant="body2" color={'error'} padding={1} fontWeight={700}>
                {error.message}
            </Typography>
        </div>
    );
};
