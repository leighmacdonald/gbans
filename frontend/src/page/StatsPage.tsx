import React, { JSX } from 'react';
import Grid from '@mui/material/Unstable_Grid2';

import { WeaponsOverallContainer } from '../component/WeaponsOverallContainer';
import { MapUsageContainer } from '../component/MapUsageContainer';
import { PlayersOverallContainer } from '../component/PlayersOverallContainer';

export const StatsPage = (): JSX.Element => {
    return (
        <Grid container spacing={2}>
            <Grid xs={12}>
                <PlayersOverallContainer />
            </Grid>
            <Grid xs={12}>
                <WeaponsOverallContainer />
            </Grid>
            <Grid xs={12}>
                <MapUsageContainer />
            </Grid>
        </Grid>
    );
};
