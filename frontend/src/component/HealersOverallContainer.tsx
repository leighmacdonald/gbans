import { ChangeEvent, useState } from 'react';
import HealthAndSafetyIcon from '@mui/icons-material/HealthAndSafety';
import { HealingOverallResult } from '../api';
import { useHealerOverallStats } from '../hooks/useHealerOverallStats';
import { Order, RowsPerPage } from '../util/table.ts';
import {
    defaultFloatFmt,
    defaultFloatFmtPct,
    humanCount
} from '../util/text.tsx';
import { ContainerWithHeader } from './ContainerWithHeader';
import FmtWhenGt from './FmtWhenGT.tsx';
import { LoadingPlaceholder } from './LoadingPlaceholder';
import { PersonCell } from './PersonCell';
import { LazyTable } from './table/LazyTable';

export const HealersOverallContainer = () => {
    const [page, setPage] = useState(0);
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [rows, setRows] = useState<RowsPerPage>(RowsPerPage.TwentyFive);
    const [sortColumn, setSortColumn] =
        useState<keyof HealingOverallResult>('healing');

    const { data, loading, count } = useHealerOverallStats({
        offset: page * rows,
        limit: rows,
        order_by: sortColumn,
        desc: sortOrder == 'desc'
    });

    return (
        <ContainerWithHeader
            title={'Top 250 Medic By Healing'}
            iconLeft={<HealthAndSafetyIcon />}
        >
            {loading ? (
                <LoadingPlaceholder />
            ) : (
                <LazyTable<HealingOverallResult>
                    showPager={true}
                    count={count}
                    rows={data}
                    page={Number(page ?? 0)}
                    rowsPerPage={rows}
                    sortOrder={sortOrder}
                    sortColumn={sortColumn}
                    onSortColumnChanged={async (column) => {
                        setSortColumn(column);
                    }}
                    onSortOrderChanged={async (direction) => {
                        setSortOrder(direction);
                    }}
                    onPageChange={(_, newPage: number) => {
                        setPage(newPage);
                    }}
                    onRowsPerPageChange={(
                        event: ChangeEvent<
                            HTMLInputElement | HTMLTextAreaElement
                        >
                    ) => {
                        setRows(Number(event.target.value));
                        setPage(0);
                    }}
                    columns={[
                        {
                            label: '#',
                            sortable: true,
                            sortKey: 'rank',
                            align: 'center',
                            tooltip: 'Overall Rank By Healing',
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
                                FmtWhenGt(obj.healing, humanCount)
                        },
                        {
                            label: 'A',
                            sortable: true,
                            sortKey: 'assists',
                            tooltip: 'Total Assists',
                            renderer: (obj) =>
                                FmtWhenGt(obj.assists, humanCount)
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
                            renderer: (obj) =>
                                FmtWhenGt(obj.kad, defaultFloatFmt)
                        },
                        {
                            label: 'HPM',
                            sortable: true,
                            sortKey: 'hpm',
                            tooltip: 'Overall Healing Per Minute',
                            renderer: (obj) =>
                                FmtWhenGt(obj.hpm, () =>
                                    defaultFloatFmt(obj.hpm)
                                )
                        },
                        {
                            label: 'DT',
                            sortable: true,
                            sortKey: 'damage_taken',
                            tooltip: 'Total Damage Taken',
                            renderer: (obj) =>
                                FmtWhenGt(obj.damage_taken, humanCount)
                        },
                        {
                            label: 'DTM',
                            sortable: true,
                            sortKey: 'dtm',
                            tooltip: 'Overall Damage Taken Per Minute',
                            renderer: (obj) =>
                                FmtWhenGt(obj.dtm, () =>
                                    defaultFloatFmt(obj.dtm)
                                )
                        },
                        {
                            label: 'DM',
                            sortable: true,
                            sortKey: 'dominations',
                            tooltip: 'Total Dominations',
                            renderer: (obj) =>
                                FmtWhenGt(obj.dominations, humanCount)
                        },
                        {
                            label: 'Dr',
                            sortable: true,
                            sortKey: 'drops',
                            tooltip: 'Total Drops',
                            renderer: (obj) => FmtWhenGt(obj.drops, humanCount)
                        },
                        {
                            label: 'Ub',
                            sortable: true,
                            sortKey: 'charges_uber',
                            tooltip: 'Total Uber Charges',
                            renderer: (obj) =>
                                FmtWhenGt(obj.charges_uber, humanCount)
                        },
                        {
                            label: 'Kr',
                            sortable: true,
                            sortKey: 'charges_kritz',
                            tooltip: 'Total Kritz Charges',
                            renderer: (obj) =>
                                FmtWhenGt(obj.charges_kritz, humanCount)
                        },
                        {
                            label: 'Qf',
                            sortable: true,
                            sortKey: 'charges_quickfix',
                            tooltip: 'Total Quick-Fix Charges',
                            renderer: (obj) =>
                                FmtWhenGt(obj.charges_quickfix, humanCount)
                        },
                        {
                            label: 'Va',
                            sortable: true,
                            sortKey: 'charges_vacc',
                            tooltip: 'Total Vaccinator Charges',
                            renderer: (obj) =>
                                FmtWhenGt(obj.charges_vacc, humanCount)
                        },
                        {
                            label: 'WR',
                            sortable: true,
                            sortKey: 'win_rate',
                            tooltip: 'Win Rate %',
                            renderer: (obj) =>
                                FmtWhenGt(obj.win_rate, defaultFloatFmtPct)
                        }
                    ]}
                />
            )}
        </ContainerWithHeader>
    );
};
