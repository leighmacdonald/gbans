import InfoIcon from '@mui/icons-material/Info';
import WarningIcon from '@mui/icons-material/Warning';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useWarningState } from '../hooks/useWarningState';
import { ContainerWithHeader } from './ContainerWithHeader';
import { LoadingPlaceholder } from './LoadingPlaceholder';
import { WarningStateTable } from './table/WarningStateTable';

export const WarningStateContainer = () => {
    const { data, loading } = useWarningState();

    return (
        <Stack spacing={2}>
            <ContainerWithHeader title={`Current Warning State (Max Weight: ${data.max_weight})`} iconLeft={<WarningIcon />}>
                {loading ? <LoadingPlaceholder /> : <WarningStateTable warnings={data.current} />}
            </ContainerWithHeader>
            <ContainerWithHeader title={'How it works'} iconLeft={<InfoIcon />}>
                <Typography variant={'body1'}>
                    The way the warning tracking works is that each time a user triggers a match, it gets a entry in the table based on the
                    weight of the match. The individual match weight is determined by the word filter defined above. Once the sum of their
                    triggers exceeds the max weight the user will have action taken against them automatically. Matched entries are
                    ephemeral and are removed over time based on the configured timeout value.
                </Typography>
            </ContainerWithHeader>
        </Stack>
    );
};
