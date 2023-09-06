import React, { JSX } from 'react';
import Grid from '@mui/material/Unstable_Grid2';
import InsightsIcon from '@mui/icons-material/Insights';
import { MapUsageContainer } from '../component/MapUsageContainer';
import { PlayersOverallContainer } from '../component/PlayersOverallContainer';
import { WeaponsStatListContainer } from '../component/WeaponsStatListContainer';
import { apiGetWeaponsOverall } from '../api';

export const StatsPage = (): JSX.Element => {
    return (
        <Grid container spacing={2}>
            <Grid xs={12}>
                <PlayersOverallContainer />
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
