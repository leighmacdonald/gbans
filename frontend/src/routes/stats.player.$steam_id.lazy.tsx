import { useParams } from 'react-router-dom';
import Grid from '@mui/material/Unstable_Grid2';
import { createLazyFileRoute } from '@tanstack/react-router';
import { PlayerClassStatsContainer } from '../component/PlayerClassStatsContainer';
import { PageNotFoundLazy } from './pageNotFound.lazy.tsx';

export const Route = createLazyFileRoute('/stats/player/$steam_id')({
    component: PlayerStats
});

function PlayerStats() {
    const { steam_id } = useParams();

    if (!steam_id) {
        return <PageNotFoundLazy error={'Invalid steam id'} />;
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
