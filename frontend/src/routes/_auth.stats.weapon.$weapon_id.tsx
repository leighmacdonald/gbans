import { useState } from 'react';
import InsightsIcon from '@mui/icons-material/Insights';
import Grid from '@mui/material/Grid2';
import Typography from '@mui/material/Typography';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';
import { createColumnHelper, getCoreRowModel, getPaginationRowModel, useReactTable } from '@tanstack/react-table';
import { apiGetPlayerWeaponStats, PlayerWeaponStats, PlayerWeaponStatsResponse } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { DataTable } from '../component/DataTable.tsx';
import FmtWhenGt from '../component/FmtWhenGT.tsx';
import { PaginatorLocal } from '../component/PaginatorLocal.tsx';
import { PersonCell } from '../component/PersonCell';
import { TableCellSmall } from '../component/TableCellSmall.tsx';
import { Title } from '../component/Title';
import { RowsPerPage } from '../util/table.ts';
import { defaultFloatFmtPct, humanCount } from '../util/text.tsx';

export const Route = createFileRoute('/_auth/stats/weapon/$weapon_id')({
    component: StatsWeapon
});

function StatsWeapon() {
    const { weapon_id } = Route.useParams();
    const { data, isLoading } = useQuery({
        queryKey: ['statsWeapons', { weapon_id }],
        queryFn: async () => apiGetPlayerWeaponStats(Number(weapon_id))
    });

    return (
        <Grid container spacing={2}>
            {data?.weapon?.name ? <Title>{data?.weapon?.name}</Title> : null}
            <Grid size={{ xs: 12 }}>
                <ContainerWithHeader
                    title={`Top 250 Weapon Users: ${isLoading ? 'Loading...' : data?.weapon?.name}`}
                    iconLeft={<InsightsIcon />}
                >
                    <StatsWeapons
                        stats={data ?? { data: [], weapon: { weapon_id: 0, name: '', key: '' }, count: 0 }}
                        isLoading={isLoading}
                    />
                </ContainerWithHeader>
            </Grid>
        </Grid>
    );
}

const columnHelper = createColumnHelper<PlayerWeaponStats>();

const StatsWeapons = ({ stats, isLoading }: { stats: PlayerWeaponStatsResponse; isLoading: boolean }) => {
    const [pagination, setPagination] = useState({
        pageIndex: 0, //initial page index
        pageSize: RowsPerPage.TwentyFive //default page size
    });

    const columns = [
        columnHelper.accessor('rank', {
            header: '#',
            size: 40,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{info.getValue()}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('steam_id', {
            header: 'Name',
            size: 400,
            cell: (info) => (
                <TableCellSmall>
                    <PersonCell
                        steam_id={stats.data[info.row.index].steam_id}
                        personaname={stats.data[info.row.index].personaname}
                        avatar_hash={stats.data[info.row.index].avatar_hash}
                    />
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('kills', {
            header: 'Kills',
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
                </TableCellSmall>
            )
        }),

        columnHelper.accessor('damage', {
            header: 'Kills%',
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), defaultFloatFmtPct)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('shots', {
            header: 'Shots',
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('hits', {
            header: 'Hits',
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
                </TableCellSmall>
            )
        }),

        columnHelper.accessor('accuracy', {
            header: 'Acc%',
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), () => defaultFloatFmtPct(info.getValue()))}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('airshots', {
            header: 'As',
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
                </TableCellSmall>
            )
        }),

        columnHelper.accessor('backstabs', {
            header: 'Bs',
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
                </TableCellSmall>
            )
        }),

        columnHelper.accessor('headshots', {
            header: 'Hs',
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
                </TableCellSmall>
            )
        })
    ];

    const table = useReactTable({
        data: stats.data,
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        getPaginationRowModel: getPaginationRowModel(),
        onPaginationChange: setPagination,
        state: {
            pagination
        }
    });

    return (
        <>
            <DataTable table={table} isLoading={isLoading} />
            <PaginatorLocal
                onRowsChange={(rows) => {
                    setPagination((prev) => {
                        return { ...prev, pageSize: rows };
                    });
                }}
                onPageChange={(page) => {
                    setPagination((prev) => {
                        return { ...prev, pageIndex: page };
                    });
                }}
                count={stats.count}
                rows={pagination.pageSize}
                page={pagination.pageIndex}
            />
        </>
    );
};
