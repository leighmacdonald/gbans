import React, { useState } from 'react';
import InsightsIcon from '@mui/icons-material/Insights';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { PlayerWeaponStats } from '../api';
import { usePlayersOverallStats } from '../hooks/usePlayersOverallStats';
import { defaultFloatFmt, defaultFloatFmtPct, humanCount } from '../util/text';
import { ContainerWithHeader } from './ContainerWithHeader';
import { LoadingPlaceholder } from './LoadingPlaceholder';
import { PersonCell } from './PersonCell';
import { LazyTable, Order, RowsPerPage } from './table/LazyTable';

export const fmtWhenGt = (
    value: number,
    fmt?: (value: number) => string,
    gt: number = 0,
    fallback: string = ''
) => {
    return value > 1000 ? (
        <Tooltip title={`${value}`}>
            <Typography
                variant={'body1'}
                padding={0}
                sx={{ fontFamily: 'Monospace' }}
            >
                {value > gt ? (fmt ? fmt(value) : `${value}`) : fallback}
            </Typography>
        </Tooltip>
    ) : (
        <Typography
            variant={'body1'}
            padding={0}
            sx={{ fontFamily: 'Monospace' }}
        >
            {value > gt ? (fmt ? fmt(value) : `${value}`) : fallback}
        </Typography>
    );
};

export const PlayersOverallContainer = () => {
    const [page, setPage] = useState(0);
    const [sortOrder, setSortOrder] = useState<Order>('asc');
    const [rows, setRows] = useState<RowsPerPage>(RowsPerPage.TwentyFive);
    const [sortColumn, setSortColumn] =
        useState<keyof PlayerWeaponStats>('rank');

    const { data, loading, count } = usePlayersOverallStats({
        offset: page * rows,
        limit: rows,
        order_by: sortColumn,
        desc: sortOrder == 'desc'
    });

    return (
        <ContainerWithHeader
            title={'Top 1000 Players By Kills'}
            iconLeft={<InsightsIcon />}
        >
            {loading ? (
                <LoadingPlaceholder />
            ) : (
                <LazyTable<PlayerWeaponStats>
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
                        event: React.ChangeEvent<
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
                            label: 'KA',
                            sortable: true,
                            sortKey: 'ka',
                            tooltip: 'Total Kills + Assists',
                            renderer: (obj) => fmtWhenGt(obj.ka, humanCount)
                        },
                        {
                            label: 'K',
                            sortable: true,
                            sortKey: 'kills',
                            tooltip: 'Total Kills',
                            renderer: (obj) => fmtWhenGt(obj.kills, humanCount)
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
                            renderer: (obj) => fmtWhenGt(obj.deaths, humanCount)
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
                            label: 'Sht',
                            sortable: true,
                            sortKey: 'shots',
                            tooltip: 'Total Shots',
                            renderer: (obj) => fmtWhenGt(obj.shots, humanCount)
                        },
                        {
                            label: 'Hit',
                            sortable: true,
                            sortKey: 'hits',
                            tooltip: 'Total Hits',
                            renderer: (obj) => fmtWhenGt(obj.hits, humanCount)
                        },
                        {
                            label: 'Acc%',
                            sortable: true,
                            sortKey: 'accuracy',
                            tooltip: 'Overall Accuracy',
                            renderer: (obj) =>
                                fmtWhenGt(obj.accuracy, () =>
                                    defaultFloatFmtPct(obj.accuracy)
                                )
                        },
                        {
                            label: 'A',
                            sortable: true,
                            sortKey: 'airshots',
                            tooltip: 'Total Airshots',
                            renderer: (obj) =>
                                fmtWhenGt(obj.airshots, humanCount)
                        },
                        {
                            label: 'B',
                            sortable: true,
                            sortKey: 'backstabs',
                            tooltip: 'Total Backstabs',
                            renderer: (obj) =>
                                fmtWhenGt(obj.backstabs, humanCount)
                        },
                        {
                            label: 'H',
                            sortable: true,
                            sortKey: 'headshots',
                            tooltip: 'Total Headshots',
                            renderer: (obj) =>
                                fmtWhenGt(obj.headshots, humanCount)
                        },
                        {
                            label: 'Dmg',
                            sortable: true,
                            sortKey: 'damage',
                            tooltip: 'Total Damage',
                            renderer: (obj) => fmtWhenGt(obj.damage, humanCount)
                        },
                        {
                            label: 'DPM',
                            sortable: true,
                            sortKey: 'dpm',
                            tooltip: 'Overall Damage Per Minute',
                            renderer: (obj) =>
                                fmtWhenGt(obj.shots, () =>
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
                            label: 'CP',
                            sortable: true,
                            sortKey: 'captures',
                            tooltip: 'Total Captures',
                            renderer: (obj) =>
                                fmtWhenGt(obj.captures, humanCount)
                        }
                    ]}
                />
            )}
        </ContainerWithHeader>
    );
};
