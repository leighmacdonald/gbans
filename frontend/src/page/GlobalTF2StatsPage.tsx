import React, { useEffect, useState } from 'react';
import Grid from '@mui/material/Grid';
import Paper from '@mui/material/Paper';
import { Heading } from '../component/Heading';
import { apiGetGlobalTF2Stats, GlobalTF2StatSnapshot } from '../api';
import { PlayerStatsChart } from '../component/PlayerStatsChart';
import { ServerStatsChart } from '../component/ServerStatsChart';

export const GlobalTF2StatsPage = (): JSX.Element => {
    const [data, setData] = useState<GlobalTF2StatSnapshot[]>([]);

    useEffect(() => {
        apiGetGlobalTF2Stats().then((resp) => {
            if (!resp.status) {
                return;
            }
            setData(resp.result ?? []);
        });
    }, []);

    return (
        <Grid container spacing={3} paddingTop={3}>
            <Grid item xs={12}>
                <Paper elevation={1}>
                    <Heading>Global TF2 Player Stats</Heading>
                    <PlayerStatsChart data={data} />
                </Paper>
            </Grid>
            <Grid item xs={12}>
                <Paper elevation={1}>
                    <Heading>Global TF2 Server Stats</Heading>
                    <ServerStatsChart data={data} />
                </Paper>
            </Grid>
        </Grid>
    );
};
