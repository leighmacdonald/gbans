import InsightsIcon from '@mui/icons-material/Insights';
import Grid from '@mui/material/Unstable_Grid2';
import React, { JSX } from 'react';
import { apiGetWeaponsOverall } from '../api';
import { HealersOverallContainer } from '../component/HealersOverallContainer';
import { MapUsageContainer } from '../component/MapUsageContainer';
import { PlayersOverallContainer } from '../component/PlayersOverallContainer';
import { WeaponsStatListContainer } from '../component/WeaponsStatListContainer';

export const StatsPage = (): JSX.Element => {
    return (
        <Grid container spacing={2}>
            <Grid xs={12}>
                <PlayersOverallContainer />
            </Grid>
            <Grid xs={12}>
                <HealersOverallContainer />
            </Grid>
            <Grid xs={12}>
                <WeaponsStatListContainer
                    title={'Overall Weapon Stats'}
                    icon={<InsightsIcon />}
                    fetchData={() => apiGetWeaponsOverall()}
                />
            </Grid>
            <Grid xs={12}>
                <MapUsageContainer />
            </Grid>
        </Grid>
    );
};
