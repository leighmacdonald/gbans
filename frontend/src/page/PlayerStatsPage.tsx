import React, { JSX } from 'react';
import Grid from '@mui/material/Unstable_Grid2';
import { useParams } from 'react-router-dom';
import { PageNotFound } from './PageNotFound';
import { PlayerClassStatsContainer } from '../component/PlayerClassStatsContainer';

export const PlayerStatsPage = (): JSX.Element => {
    const { steam_id } = useParams();

    // eslint-disable-next-line @typescript-eslint/no-unused-vars

    if (!steam_id) {
        return <PageNotFound error={'Invalid steam id'} />;
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
