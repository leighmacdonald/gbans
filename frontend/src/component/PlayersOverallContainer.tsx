import React, { useEffect, useMemo, useState } from 'react';
import { apiGetPlayersOverall, PlayerWeaponStats } from '../api';
import { compare, Order, RowsPerPage, stableSort } from './DataTable';
import { ContainerWithHeader } from './ContainerWithHeader';
import InsightsIcon from '@mui/icons-material/Insights';
import Grid from '@mui/material/Unstable_Grid2';
import { LoadingSpinner } from './LoadingSpinner';
import Stack from '@mui/material/Stack';
import Pagination from '@mui/material/Pagination';
import { LazyTable } from './LazyTable';
import { defaultFloatFmt, defaultFloatFmtPct, humanCount } from '../util/text';
import { PersonCell } from './PersonCell';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';

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
    const [details, setDetails] = useState<PlayerWeaponStats[]>([]);
    const [loading, setLoading] = useState(true);
    const [sortOrder, setSortOrder] = useState<Order>('asc');
    const [sortColumn, setSortColumn] =
        useState<keyof PlayerWeaponStats>('rank');
    const [page, setPage] = useState(1);

    useEffect(() => {
        apiGetPlayersOverall()
            .then((resp) => {
                if (resp.result) {
                    setDetails(resp.result);
                }
            })
            .finally(() => {
                setLoading(false);
            });
    }, []);

    const rows = useMemo(() => {
        return stableSort(details ?? [], compare(sortOrder, sortColumn)).slice(
            (page - 1) * RowsPerPage.TwentyFive,
            (page - 1) * RowsPerPage.TwentyFive + RowsPerPage.TwentyFive
        );
    }, [details, page, sortColumn, sortOrder]);

    return (
        <ContainerWithHeader
            title={'Top 1000 Players By Kills'}
            iconLeft={<InsightsIcon />}
        >
            <Grid container>
                <Grid xs={12}>
                    {loading ? (
                        <LoadingSpinner />
                    ) : (
                        <Stack>
                            <Stack direction={'row-reverse'}>
                                <Pagination
                                    page={page}
                                    count={Math.ceil(details.length / 25)}
                                    showFirstButton
                                    showLastButton
                                    onChange={(_, newPage) => {
                                        setPage(newPage);
                                    }}
                                />
                            </Stack>
                            <LazyTable<PlayerWeaponStats>
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
                                                    avatar_hash={
                                                        obj.avatar_hash
                                                    }
                                                    personaname={
                                                        obj.personaname
                                                    }
                                                />
                                            );
                                        }
                                    },
                                    {
                                        label: 'KA',
                                        sortable: true,
                                        sortKey: 'ka',
                                        tooltip: 'Total Kills + Assists',
                                        renderer: (obj) =>
                                            fmtWhenGt(obj.ka, humanCount)
                                    },
                                    {
                                        label: 'K',
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
                                        label: 'Sht',
                                        sortable: true,
                                        sortKey: 'shots',
                                        tooltip: 'Total Shots',
                                        renderer: (obj) =>
                                            fmtWhenGt(obj.shots, humanCount)
                                    },
                                    {
                                        label: 'Hit',
                                        sortable: true,
                                        sortKey: 'hits',
                                        tooltip: 'Total Hits',
                                        renderer: (obj) =>
                                            fmtWhenGt(obj.hits, humanCount)
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
                                        label: 'As',
                                        sortable: true,
                                        sortKey: 'airshots',
                                        tooltip: 'Total Airshots',
                                        renderer: (obj) =>
                                            fmtWhenGt(obj.airshots, humanCount)
                                    },
                                    {
                                        label: 'Bs',
                                        sortable: true,
                                        sortKey: 'backstabs',
                                        tooltip: 'Total Backstabs',
                                        renderer: (obj) =>
                                            fmtWhenGt(obj.backstabs, humanCount)
                                    },
                                    {
                                        label: 'Hs',
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
                                        renderer: (obj) =>
                                            fmtWhenGt(obj.damage, humanCount)
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
                                        label: 'DmgT',
                                        sortable: true,
                                        sortKey: 'damage_taken',
                                        tooltip: 'Total Damage Taken',
                                        renderer: (obj) =>
                                            fmtWhenGt(
                                                obj.damage_taken,
                                                humanCount
                                            )
                                    },
                                    {
                                        label: 'Dom',
                                        sortable: true,
                                        sortKey: 'dominations',
                                        tooltip: 'Total Dominations',
                                        renderer: (obj) =>
                                            fmtWhenGt(obj.damage, humanCount)
                                    },
                                    {
                                        label: 'Cap',
                                        sortable: true,
                                        sortKey: 'captures',
                                        tooltip: 'Total Captures',
                                        renderer: (obj) =>
                                            fmtWhenGt(obj.captures, humanCount)
                                    }
                                ]}
                                sortOrder={sortOrder}
                                sortColumn={sortColumn}
                                onSortColumnChanged={async (column) => {
                                    setSortColumn(column);
                                }}
                                onSortOrderChanged={async (direction) => {
                                    setSortOrder(direction);
                                }}
                                rows={rows}
                            />
                        </Stack>
                    )}
                </Grid>
            </Grid>
        </ContainerWithHeader>
    );
};
