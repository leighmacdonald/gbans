import React from 'react';
import {
    apiGetPlayerClassOverallStats,
    PlayerClassOverallResult
} from '../api';
import { ContainerWithHeader } from './ContainerWithHeader';
import Grid from '@mui/material/Unstable_Grid2';
import { LazyTableSimple } from './LazyTableSimple';
import { fmtWhenGt } from './PlayersOverallContainer';
import { defaultFloatFmt, humanCount } from '../util/text';
import BarChartIcon from '@mui/icons-material/BarChart';
import { PlayerClassImg } from './PlayerClassImg';
import { formatDistance } from 'date-fns';

interface PlayerClassStatsContainerProps {
    steam_id: string;
}

export const PlayerClassStatsContainer = ({
    steam_id
}: PlayerClassStatsContainerProps) => {
    return (
        <ContainerWithHeader
            title={'Overall Player Class Stats'}
            iconLeft={<BarChartIcon />}
        >
            <Grid container>
                <Grid xs={12}>
                    <LazyTableSimple<PlayerClassOverallResult>
                        paged={false}
                        showPager={false}
                        fetchData={() =>
                            apiGetPlayerClassOverallStats(steam_id)
                        }
                        columns={[
                            {
                                label: 'Class',
                                sortable: true,
                                sortKey: 'player_class_id',
                                tooltip: 'Player Class',
                                align: 'center',
                                renderer: (obj) => {
                                    return (
                                        <PlayerClassImg
                                            cls={obj.player_class_id}
                                        />
                                    );
                                }
                            },
                            {
                                label: 'Playtime',
                                sortable: true,
                                sortKey: 'playtime',
                                tooltip: 'Total Playtime',
                                renderer: (obj) =>
                                    `${formatDistance(0, obj.playtime * 1000, {
                                        includeSeconds: true
                                    })}`
                            },
                            {
                                label: 'KA',
                                sortable: true,
                                sortKey: 'ka',
                                tooltip: 'Total Kills + Assists',
                                renderer: (obj) => fmtWhenGt(obj.ka, humanCount)
                            },
                            {
                                label: 'Kills',
                                sortable: true,
                                sortKey: 'kills',
                                tooltip: 'Total Kills',
                                renderer: (obj) =>
                                    fmtWhenGt(obj.kills, humanCount)
                            },
                            {
                                label: 'A',
                                sortable: true,
                                sortKey: 'assists',
                                tooltip: 'Total Assists',
                                renderer: (obj) =>
                                    fmtWhenGt(obj.assists, humanCount)
                            },
                            {
                                label: 'D',
                                sortable: true,
                                sortKey: 'deaths',
                                tooltip: 'Total Deaths',
                                renderer: (obj) =>
                                    fmtWhenGt(obj.deaths, humanCount)
                            },
                            {
                                label: 'KAD',
                                sortable: true,
                                sortKey: 'kad',
                                tooltip: 'Kills+Assists:Deaths Ratio',
                                renderer: (obj) =>
                                    fmtWhenGt(obj.kad, defaultFloatFmt)
                            },
                            {
                                label: 'Dmg',
                                sortable: true,
                                sortKey: 'damage',
                                tooltip: 'Total Damage',
                                renderer: (obj) =>
                                    fmtWhenGt(obj.damage, humanCount)
                            },
                            {
                                label: 'DPM',
                                sortable: true,
                                sortKey: 'dpm',
                                tooltip: 'Overall Damage Per Minute',
                                renderer: (obj) =>
                                    fmtWhenGt(obj.dpm, () =>
                                        defaultFloatFmt(obj.dpm)
                                    )
                            },
                            {
                                label: 'DT',
                                sortable: true,
                                sortKey: 'damage_taken',
                                tooltip: 'Total Damage Taken',
                                renderer: (obj) =>
                                    fmtWhenGt(obj.damage_taken, humanCount)
                            },
                            {
                                label: 'DM',
                                sortable: true,
                                sortKey: 'dominations',
                                tooltip: 'Total Dominations',
                                renderer: (obj) =>
                                    fmtWhenGt(obj.dominations, humanCount)
                            },
                            {
                                label: 'DD',
                                sortable: true,
                                sortKey: 'dominated',
                                tooltip: 'Total Times Dominated',
                                renderer: (obj) =>
                                    fmtWhenGt(obj.dominated, humanCount)
                            },
                            {
                                label: 'RV',
                                sortable: true,
                                sortKey: 'revenges',
                                tooltip: 'Total Revenges',
                                renderer: (obj) =>
                                    fmtWhenGt(obj.revenges, humanCount)
                            },
                            {
                                label: 'CP',
                                sortable: true,
                                sortKey: 'captures',
                                tooltip: 'Total Captures',
                                renderer: (obj) =>
                                    fmtWhenGt(obj.captures, humanCount)
                            }
                        ]}
                        defaultSortColumn={'ka'}
                    />
                </Grid>
            </Grid>
        </ContainerWithHeader>
    );
};
