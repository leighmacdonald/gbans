import BarChartIcon from '@mui/icons-material/BarChart';
import Grid from '@mui/material/Unstable_Grid2';
import { formatDistance } from 'date-fns';
import { apiGetPlayerClassOverallStats, PlayerClassOverallResult } from '../api';
import { defaultFloatFmt, humanCount } from '../util/text.tsx';
import { ContainerWithHeader } from './ContainerWithHeader';
import FmtWhenGt from './FmtWhenGT.tsx';
import { PlayerClassImg } from './PlayerClassImg';
import { LazyTableSimple } from './table/LazyTableSimple';

interface PlayerClassStatsContainerProps {
    steam_id: string;
}

export const PlayerClassStatsContainer = ({ steam_id }: PlayerClassStatsContainerProps) => {
    return (
        <ContainerWithHeader title={'Player Overall Stats By Class'} iconLeft={<BarChartIcon />}>
            <Grid container>
                <Grid xs={12}>
                    <LazyTableSimple<PlayerClassOverallResult>
                        paged={false}
                        showPager={false}
                        fetchData={() => apiGetPlayerClassOverallStats(steam_id)}
                        columns={[
                            {
                                label: 'Class',
                                sortable: true,
                                sortKey: 'player_class_id',
                                tooltip: 'Player Class',
                                align: 'center',
                                renderer: (obj) => {
                                    return <PlayerClassImg cls={obj.player_class_id} />;
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
                                renderer: (obj) => FmtWhenGt(obj.ka, humanCount)
                            },
                            {
                                label: 'Kills',
                                sortable: true,
                                sortKey: 'kills',
                                tooltip: 'Total Kills',
                                renderer: (obj) => FmtWhenGt(obj.kills, humanCount)
                            },
                            {
                                label: 'A',
                                sortable: true,
                                sortKey: 'assists',
                                tooltip: 'Total Assists',
                                renderer: (obj) => FmtWhenGt(obj.assists, humanCount)
                            },
                            {
                                label: 'D',
                                sortable: true,
                                sortKey: 'deaths',
                                tooltip: 'Total Deaths',
                                renderer: (obj) => FmtWhenGt(obj.deaths, humanCount)
                            },
                            {
                                label: 'KAD',
                                sortable: true,
                                sortKey: 'kad',
                                tooltip: 'Kills+Assists:Deaths Ratio',
                                renderer: (obj) => FmtWhenGt(obj.kad, defaultFloatFmt)
                            },
                            {
                                label: 'Dmg',
                                sortable: true,
                                sortKey: 'damage',
                                tooltip: 'Total Damage',
                                renderer: (obj) => FmtWhenGt(obj.damage, humanCount)
                            },
                            {
                                label: 'DPM',
                                sortable: true,
                                sortKey: 'dpm',
                                tooltip: 'Overall Damage Per Minute',
                                renderer: (obj) => FmtWhenGt(obj.dpm, () => defaultFloatFmt(obj.dpm))
                            },
                            {
                                label: 'DT',
                                sortable: true,
                                sortKey: 'damage_taken',
                                tooltip: 'Total Damage Taken',
                                renderer: (obj) => FmtWhenGt(obj.damage_taken, humanCount)
                            },
                            {
                                label: 'DM',
                                sortable: true,
                                sortKey: 'dominations',
                                tooltip: 'Total Dominations',
                                renderer: (obj) => FmtWhenGt(obj.dominations, humanCount)
                            },
                            {
                                label: 'DD',
                                sortable: true,
                                sortKey: 'dominated',
                                tooltip: 'Total Times Dominated',
                                renderer: (obj) => FmtWhenGt(obj.dominated, humanCount)
                            },
                            {
                                label: 'RV',
                                sortable: true,
                                sortKey: 'revenges',
                                tooltip: 'Total Revenges',
                                renderer: (obj) => FmtWhenGt(obj.revenges, humanCount)
                            },
                            {
                                label: 'CP',
                                sortable: true,
                                sortKey: 'captures',
                                tooltip: 'Total Captures',
                                renderer: (obj) => FmtWhenGt(obj.captures, humanCount)
                            }
                        ]}
                        defaultSortColumn={'ka'}
                    />
                </Grid>
            </Grid>
        </ContainerWithHeader>
    );
};
