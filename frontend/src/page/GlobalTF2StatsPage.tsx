import React, { useEffect, useState } from 'react';
import Grid from '@mui/material/Grid';
import Paper from '@mui/material/Paper';
import { Heading } from '../component/Heading';
import {
    apiGetGlobalTF2Stats,
    GlobalTF2StatSnapshot,
    StatDuration,
    statDurationString
} from '../api';
import { PlayerStatsChart } from '../component/PlayerStatsChart';
import { ServerStatsChart } from '../component/ServerStatsChart';
import Select from '@mui/material/Select';
import MenuItem from '@mui/material/MenuItem';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import Stack from '@mui/material/Stack';

export const GlobalTF2StatsPage = (): JSX.Element => {
    const [data, setData] = useState<GlobalTF2StatSnapshot[]>([]);
    const [duration, setDuration] = useState<StatDuration>(StatDuration.Hourly);

    useEffect(() => {
        apiGetGlobalTF2Stats(duration).then((resp) => {
            if (!resp.status) {
                return;
            }
            setData(resp.result ?? []);
        });
    }, [duration]);
    const durations = [StatDuration.Hourly, StatDuration.Daily];
    return (
        <Grid container paddingTop={3}>
            <Grid item xs={12}>
                <Stack spacing={2}>
                    <Paper elevation={1}>
                        <FormControl fullWidth>
                            <InputLabel id="interval-label">
                                Graph Interval
                            </InputLabel>
                            <Select<StatDuration>
                                labelId={'interval-label'}
                                id={'interval'}
                                label={'Graph Interval'}
                                value={duration}
                                onChange={(evt) => {
                                    setDuration(
                                        evt.target.value as StatDuration
                                    );
                                }}
                            >
                                {durations.map((d) => {
                                    return (
                                        <MenuItem value={d} key={`dur-${d}`}>
                                            {statDurationString(d)}
                                        </MenuItem>
                                    );
                                })}
                            </Select>
                        </FormControl>
                    </Paper>

                    <Paper elevation={1}>
                        <Stack spacing={2}>
                            <Heading>Global TF2 Player Stats</Heading>

                            <PlayerStatsChart data={data} />
                        </Stack>
                    </Paper>

                    <Paper elevation={1}>
                        <Heading>Global TF2 Server Stats</Heading>
                        <ServerStatsChart data={data} />
                    </Paper>
                </Stack>
            </Grid>
        </Grid>
    );
};
