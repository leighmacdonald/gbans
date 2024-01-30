import React, { JSX } from 'react';
import { useParams } from 'react-router-dom';
import Grid from '@mui/material/Unstable_Grid2';
import { PlayerClassStatsContainer } from '../component/PlayerClassStatsContainer';
import { PageNotFoundPage } from './PageNotFoundPage';

export const PlayerStatsPage = (): JSX.Element => {
    const { steam_id } = useParams();

    if (!steam_id) {
        return <PageNotFoundPage error={'Invalid steam id'} />;
    }

    return (
        <Grid container>
            <Grid xs={12}>
                <PlayerClassStatsContainer steam_id={steam_id} />
            </Grid>
            <Grid xs={6}></Grid>
        </Grid>
    );
};

export default PlayerStatsPage;
