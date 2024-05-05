import Grid from '@mui/material/Unstable_Grid2';
import { createLazyFileRoute } from '@tanstack/react-router';
import { HealersOverallContainer } from '../component/HealersOverallContainer.tsx';
import { MapUsageContainer } from '../component/MapUsageContainer.tsx';
import { PlayersOverallContainer } from '../component/PlayersOverallContainer.tsx';
import { WeaponsStatListContainer } from '../component/WeaponsStatListContainer.tsx';

export const Route = createLazyFileRoute('/_auth/stats/')({
    component: Stats
});

function Stats() {
    return (
        <Grid container spacing={2}>
            <Grid xs={12}>
                <PlayersOverallContainer />
            </Grid>
            <Grid xs={12}>
                <HealersOverallContainer />
            </Grid>
            <Grid xs={12}>
                <WeaponsStatListContainer />
            </Grid>
            <Grid xs={12}>
                <MapUsageContainer />
            </Grid>
        </Grid>
    );
}
