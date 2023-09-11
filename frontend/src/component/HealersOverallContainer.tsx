import React from 'react';
import { apiGetHealersOverall, HealingOverallResult } from '../api';
import { ContainerWithHeader } from './ContainerWithHeader';
import InsightsIcon from '@mui/icons-material/Insights';
import Grid from '@mui/material/Unstable_Grid2';
import { defaultFloatFmt, defaultFloatFmtPct, humanCount } from '../util/text';
import { PersonCell } from './PersonCell';
import { LazyTableSimple } from './LazyTableSimple';
import { fmtWhenGt } from './PlayersOverallContainer';

export const HealersOverallContainer = () => {
    const fetchStats = () => apiGetHealersOverall();

    return (
        <ContainerWithHeader
            title={'Top 250 Medic By Healing'}
            iconLeft={<InsightsIcon />}
        >
            <Grid container>
                <Grid xs={12}>
                    <LazyTableSimple<HealingOverallResult>
                        fetchData={fetchStats}
                        defaultSortDir={'asc'}
                        defaultSortColumn={'rank'}
                        columns={[
                            {
                                label: '#',
                                sortable: true,
                                sortKey: 'rank',
                                align: 'center',
                                tooltip: 'Overall Rank By Kills',
                                renderer: (obj) => obj.rank
                            },
                            {
                                label: 'Name',
                                sortable: true,
                                sortKey: 'personaname',
                                tooltip: 'Name',
                                align: 'left',
                                renderer: (obj) => {
                                    return (
                                        <PersonCell
                                            steam_id={obj.steam_id}
                                            avatar_hash={obj.avatar_hash}
                                            personaname={obj.personaname}
                                        />
                                    );
                                }
                            },
                            {
                                label: 'Healing',
                                sortable: true,
                                sortKey: 'healing',
                                tooltip: 'Total Healing',
                                renderer: (obj) =>
                                    fmtWhenGt(obj.healing, humanCount)
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
                                label: 'HPM',
                                sortable: true,
                                sortKey: 'hpm',
                                tooltip: 'Overall Healing Per Minute',
                                renderer: (obj) =>
                                    fmtWhenGt(obj.hpm, () =>
                                        defaultFloatFmt(obj.hpm)
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
                                label: 'DTM',
                                sortable: true,
                                sortKey: 'dtm',
                                tooltip: 'Overall Damage Taken Per Minute',
                                renderer: (obj) =>
                                    fmtWhenGt(obj.dtm, () =>
                                        defaultFloatFmt(obj.dtm)
                                    )
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
                                label: 'Dr',
                                sortable: true,
                                sortKey: 'drops',
                                tooltip: 'Total Drops',
                                renderer: (obj) =>
                                    fmtWhenGt(obj.drops, humanCount)
                            },
                            {
                                label: 'Ub',
                                sortable: true,
                                sortKey: 'charges_uber',
                                tooltip: 'Total Uber Charges',
                                renderer: (obj) =>
                                    fmtWhenGt(obj.charges_uber, humanCount)
                            },
                            {
                                label: 'Kr',
                                sortable: true,
                                sortKey: 'charges_kritz',
                                tooltip: 'Total Kritz Charges',
                                renderer: (obj) =>
                                    fmtWhenGt(obj.charges_kritz, humanCount)
                            },
                            {
                                label: 'Qf',
                                sortable: true,
                                sortKey: 'charges_quickfix',
                                tooltip: 'Total Quick-Fix Charges',
                                renderer: (obj) =>
                                    fmtWhenGt(obj.charges_quickfix, humanCount)
                            },
                            {
                                label: 'Va',
                                sortable: true,
                                sortKey: 'charges_vacc',
                                tooltip: 'Total Vaccinator Charges',
                                renderer: (obj) =>
                                    fmtWhenGt(obj.charges_vacc, humanCount)
                            },
                            {
                                label: 'WR',
                                sortable: true,
                                sortKey: 'win_rate',
                                tooltip: 'Win Rate %',
                                renderer: (obj) =>
                                    fmtWhenGt(obj.win_rate, defaultFloatFmtPct)
                            }
                        ]}
                    />
                </Grid>
            </Grid>
        </ContainerWithHeader>
    );
};
