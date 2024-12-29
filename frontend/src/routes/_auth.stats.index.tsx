import Grid from '@mui/material/Grid2';
import { createFileRoute } from '@tanstack/react-router';
import { HealersOverallContainer } from '../component/HealersOverallContainer.tsx';
import { MapUsageContainer } from '../component/MapUsageContainer.tsx';
import { PlayersOverallContainer } from '../component/PlayersOverallContainer.tsx';
import { Title } from '../component/Title';
import { WeaponsStatListContainer } from '../component/WeaponsStatListContainer.tsx';

export const Route = createFileRoute('/_auth/stats/')({
    component: Stats
});

function Stats() {
    return (
        <Grid container spacing={2}>
            <Title>Stats</Title>
            <Grid size={{ xs: 12 }}>
                <PlayersOverallContainer />
            </Grid>
            <Grid size={{ xs: 12 }}>
                <HealersOverallContainer />
            </Grid>
            <Grid size={{ xs: 12 }}>
                <WeaponsStatListContainer />
            </Grid>
            <Grid size={{ xs: 12 }}>
                <MapUsageContainer />
            </Grid>
        </Grid>
    );
}
