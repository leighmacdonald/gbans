import React, { JSX } from 'react';
import Grid from '@mui/material/Unstable_Grid2';
import { useParams } from 'react-router-dom';
import { PageNotFound } from './PageNotFound';

export const PlayerStatsPage = (): JSX.Element => {
    const { steam_id } = useParams();

    if (!steam_id) {
        return <PageNotFound error={'Invalid steam id'} />;
    }

    return (
        <Grid container>
            <Grid xs={6}></Grid>
            <Grid xs={6}></Grid>
        </Grid>
    );
};
