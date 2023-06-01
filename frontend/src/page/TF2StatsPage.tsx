import React, { useEffect, useState, JSX } from 'react';
import Grid from '@mui/material/Grid';
import Select, { SelectChangeEvent } from '@mui/material/Select';
import MenuItem from '@mui/material/MenuItem';
import FormControl from '@mui/material/FormControl';
import InputLabel from '@mui/material/InputLabel';
import PeopleIcon from '@mui/icons-material/People';
import Stack from '@mui/material/Stack';
import StorageIcon from '@mui/icons-material/Storage';
import SportsIcon from '@mui/icons-material/Sports';
import SettingsSuggestIcon from '@mui/icons-material/SettingsSuggest';
import {
    apiGetTF2Stats,
    GlobalTF2StatSnapshot,
    LocalTF2StatSnapshot,
    StatDuration,
    statDurationString,
    StatSource
} from '../api';
import { PlayerStatsChart } from '../component/PlayerStatsChart';
import { ServerStatsChart } from '../component/ServerStatsChart';
import { PlayerStatsChartLocal } from '../component/PlayerStatsChartLocal';
import { ServerStatsChartLocal } from '../component/ServerStatsChartLocal';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { GameTypeStatsChartLocal } from '../component/GameTypeStatsChartLocal';
import { ServerPlayersStatsChartLocal } from '../component/ServerPlayersStatsChartLocal';

export const TF2StatsPage = (): JSX.Element => {
    const [globalData, setGlobalData] = useState<GlobalTF2StatSnapshot[]>([]);
    const [localData, setLocalData] = useState<LocalTF2StatSnapshot[]>([]);
    const [duration, setDuration] = useState<StatDuration>(StatDuration.Hourly);
    const [source, setSource] = useState<StatSource>(StatSource.Global);

    useEffect(() => {
        apiGetTF2Stats(source, duration).then((resp) => {
            if (!resp.status) {
                return;
            }
            if (source == StatSource.Global) {
                setGlobalData((resp.result as GlobalTF2StatSnapshot[]) ?? []);
            } else {
                setLocalData((resp.result as LocalTF2StatSnapshot[]) ?? []);
            }
        });
    }, [duration, source]);

    const durations = [StatDuration.Hourly];

    return (
        <Grid container paddingTop={3}>
            <Grid item xs={12}>
                <Stack spacing={2}>
                    <ContainerWithHeader
                        iconLeft={<SettingsSuggestIcon />}
                        title={'Graph Options'}
                        marginTop={0}
                    >
                        <Stack direction={'row'} spacing={2} padding={2}>
                            <FormControl fullWidth>
                                <InputLabel id="interval-label">
                                    Graph Source Type
                                </InputLabel>
                                <Select<StatSource>
                                    labelId={'source-label'}
                                    id={'source'}
                                    label={'Graph Source Type'}
                                    value={source}
                                    onChange={(
                                        evt: SelectChangeEvent<StatSource>
                                    ) => {
                                        setSource(
                                            evt.target.value as StatSource
                                        );
                                    }}
                                >
                                    <MenuItem value={StatSource.Global}>
                                        Global
                                    </MenuItem>
                                    <MenuItem value={StatSource.Local}>
                                        Local
                                    </MenuItem>
                                </Select>
                            </FormControl>
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
                                            <MenuItem
                                                value={d}
                                                key={`dur-${d}`}
                                            >
                                                {statDurationString(d)}
                                            </MenuItem>
                                        );
                                    })}
                                </Select>
                            </FormControl>
                        </Stack>
                    </ContainerWithHeader>

                    {source == StatSource.Global && (
                        <>
                            <ContainerWithHeader
                                iconLeft={<PeopleIcon />}
                                title={'Player Populations'}
                            >
                                <PlayerStatsChart data={globalData} />
                            </ContainerWithHeader>
                            <ContainerWithHeader
                                iconLeft={<StorageIcon />}
                                title={'Server Occupancy'}
                            >
                                <ServerStatsChart data={globalData} />
                            </ContainerWithHeader>
                        </>
                    )}
                    {source == StatSource.Local && (
                        <>
                            <ContainerWithHeader
                                title={'Players Per Server'}
                                iconLeft={<SportsIcon />}
                            >
                                <ServerPlayersStatsChartLocal
                                    data={localData}
                                />
                            </ContainerWithHeader>
                            <ContainerWithHeader
                                title={'Players Per Game Type'}
                                iconLeft={<SportsIcon />}
                            >
                                <GameTypeStatsChartLocal data={localData} />
                            </ContainerWithHeader>

                            <ContainerWithHeader
                                iconLeft={<PeopleIcon />}
                                title={'Player Populations'}
                            >
                                <PlayerStatsChartLocal data={localData} />
                            </ContainerWithHeader>
                            <ContainerWithHeader
                                iconLeft={<StorageIcon />}
                                title={'Server Occupancy'}
                            >
                                <ServerStatsChartLocal data={localData} />
                            </ContainerWithHeader>
                        </>
                    )}
                </Stack>
            </Grid>
        </Grid>
    );
};
