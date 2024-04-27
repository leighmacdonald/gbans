import Stack from '@mui/material/Stack';
import { createLazyFileRoute } from '@tanstack/react-router';
import { HealersOverallContainer } from '../../component/HealersOverallContainer.tsx';
import { MapUsageContainer } from '../../component/MapUsageContainer.tsx';
import { PlayersOverallContainer } from '../../component/PlayersOverallContainer.tsx';
import { WeaponsStatListContainer } from '../../component/WeaponsStatListContainer.tsx';

export const Route = createLazyFileRoute('/_auth/stats')({
    component: Stats
});

function Stats() {
    return (
        <Stack spacing={2}>
            <PlayersOverallContainer />
            <HealersOverallContainer />
            <WeaponsStatListContainer />
            <MapUsageContainer />
        </Stack>
    );
}
