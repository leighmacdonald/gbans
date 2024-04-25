import { JSX } from 'react';
import Stack from '@mui/material/Stack';
import { createLazyFileRoute } from '@tanstack/react-router';
import { HealersOverallContainer } from '../component/HealersOverallContainer';
import { MapUsageContainer } from '../component/MapUsageContainer';
import { PlayersOverallContainer } from '../component/PlayersOverallContainer';
import { WeaponsStatListContainer } from '../component/WeaponsStatListContainer';

export const Route = createLazyFileRoute('/stats')({
    component: Stats
});

export const Stats = (): JSX.Element => {
    return (
        <Stack spacing={2}>
            <PlayersOverallContainer />
            <HealersOverallContainer />
            <WeaponsStatListContainer />
            <MapUsageContainer />
        </Stack>
    );
};
