import { useState } from 'react';
import Link from '@mui/material/Link';
import Typography from '@mui/material/Typography';
import { useQuery } from '@tanstack/react-query';
import { createColumnHelper, getCoreRowModel, getPaginationRowModel, useReactTable } from '@tanstack/react-table';
import { apiGetPlayerWeaponsOverall, WeaponsOverallResult } from '../api';
import { RowsPerPage } from '../util/table.ts';
import { defaultFloatFmt, defaultFloatFmtPct, humanCount } from '../util/text.tsx';
import { DataTable } from './DataTable.tsx';
import FmtWhenGt from './FmtWhenGT.tsx';
import { PaginatorLocal } from './PaginatorLocal.tsx';
import RouterLink from './RouterLink.tsx';
import { TableCellSmall } from './TableCellSmall.tsx';

const columnHelper = createColumnHelper<WeaponsOverallResult>();

export const PlayerWeaponsStatListContainer = ({ steamId }: { steamId: string }) => {
    const [pagination, setPagination] = useState({
        pageIndex: 0, //initial page index
        pageSize: RowsPerPage.TwentyFive //default page size
    });

    const { data: stats, isLoading } = useQuery({
        queryKey: ['playerStats', { steamId }],
        queryFn: async () => {
            return await apiGetPlayerWeaponsOverall(steamId);
        }
    });

    const columns = [
        columnHelper.accessor('name', {
            header: 'Weapon',
            size: 350,
            cell: (info) => (
                <TableCellSmall>
                    <Link component={RouterLink} to={`/stats/weapon/${info.row.original.weapon_id}`}>
                        {String(info.getValue())}
                    </Link>
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
        columnHelper.accessor('kills_pct', {
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
        columnHelper.accessor('shots_pct', {
            header: 'Shot%',
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), defaultFloatFmtPct)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('hits', {
            header: 'Hits',
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), defaultFloatFmt)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('hits_pct', {
            header: 'Hits%',
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), defaultFloatFmtPct)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('accuracy', {
            header: 'Acc%',
            cell: (info) => (
                <TableCellSmall>
                    <Typography>
                        <Typography>{FmtWhenGt(info.getValue(), defaultFloatFmtPct)}</Typography>
                    </Typography>
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
        columnHelper.accessor('airshots_pct', {
            header: 'As%',
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), defaultFloatFmtPct)}</Typography>
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
        columnHelper.accessor('backstabs_pct', {
            header: 'Bs%',
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), defaultFloatFmtPct)}</Typography>
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
        }),
        columnHelper.accessor('headshots_pct', {
            header: 'Hs%',
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), defaultFloatFmtPct)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('damage', {
            header: 'Dmg',
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('damage_pct', {
            header: 'Dmg%',
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), defaultFloatFmtPct)}</Typography>
                </TableCellSmall>
            )
        })
    ];

    const table = useReactTable({
        data: stats?.data ?? [],
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        getPaginationRowModel: getPaginationRowModel(),
        onPaginationChange: setPagination, //update the pagination state when internal APIs mutate the pagination state
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
                count={stats?.count ?? 0}
                rows={pagination.pageSize}
                page={pagination.pageIndex}
            />
        </>
    );
};
