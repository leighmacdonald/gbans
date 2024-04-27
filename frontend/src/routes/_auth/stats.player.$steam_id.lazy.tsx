import Grid from '@mui/material/Unstable_Grid2';
import { createLazyFileRoute } from '@tanstack/react-router';
import { PlayerClassStatsContainer } from '../../component/PlayerClassStatsContainer.tsx';
import { PageNotFound } from '../page-not-found.lazy.tsx';

export const Route = createLazyFileRoute('/_auth/stats/player/$steam_id')({
    component: PlayerStats
});

function PlayerStats() {
    const { steam_id } = Route.useParams();

    if (!steam_id) {
        return <PageNotFound />;
    }

    return (
        <Grid container>
            <Grid xs={12}>
                <PlayerClassStatsContainer steam_id={steam_id} />
            </Grid>
            <Grid xs={6}></Grid>
        </Grid>
    );
}
