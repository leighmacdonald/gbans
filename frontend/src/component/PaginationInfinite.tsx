import { ArrowCircleLeft, ArrowCircleRight } from '@mui/icons-material';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useNavigate } from '@tanstack/react-router';
import { VCenterBox } from './VCenterBox.tsx';

export const PaginationInfinite = ({ route, page }: { route: string; page: number }) => {
    const navigate = useNavigate({ from: route });

    return (
        <Stack direction={'row'} spacing={1}>
            <IconButton
                color={'primary'}
                disabled={page <= 0}
                onClick={async () => {
                    await navigate({ search: (prev) => ({ ...prev, page: page - 1 }) });
                }}
            >
                <ArrowCircleLeft />
            </IconButton>
            <VCenterBox>
                <Typography variant={'h6'} color={'primary'}>
                    {page + 1}
                </Typography>
            </VCenterBox>
            <IconButton
                color={'primary'}
                onClick={async () => {
                    await navigate({ search: (prev) => ({ ...prev, page: page + 1 }) });
                }}
            >
                <ArrowCircleRight />
            </IconButton>
        </Stack>
    );
};
