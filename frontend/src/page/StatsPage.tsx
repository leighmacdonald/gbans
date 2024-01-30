import React, { JSX } from 'react';
import Stack from '@mui/material/Stack';
import { HealersOverallContainer } from '../component/HealersOverallContainer';
import { MapUsageContainer } from '../component/MapUsageContainer';
import { PlayersOverallContainer } from '../component/PlayersOverallContainer';
import { WeaponsStatListContainer } from '../component/WeaponsStatListContainer';

export const StatsPage = (): JSX.Element => {
    return (
        <Stack spacing={2}>
            <PlayersOverallContainer />
            <HealersOverallContainer />
            <WeaponsStatListContainer />
            <MapUsageContainer />
        </Stack>
    );
};

export default StatsPage;
