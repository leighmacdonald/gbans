import React, { JSX } from 'react';
import Grid from '@mui/material/Unstable_Grid2';
import InsightsIcon from '@mui/icons-material/Insights';
import { WeaponsStatListContainer } from '../component/WeaponsStatListContainer';
import { MapUsageContainer } from '../component/MapUsageContainer';
import { PlayersOverallContainer } from '../component/PlayersOverallContainer';
import { apiGetWeaponsOverall } from '../api';

export const StatsPage = (): JSX.Element => {
    return (
        <Grid container spacing={2}>
            <Grid xs={12}>
                <PlayersOverallContainer />
            </Grid>
            <Grid xs={12}>
                <WeaponsStatListContainer
                    fetchData={apiGetWeaponsOverall}
                    title={'Overall Weapon Stats'}
                    icon={<InsightsIcon />}
                />
            </Grid>
            <Grid xs={12}>
                <MapUsageContainer />
            </Grid>
        </Grid>
    );
};
