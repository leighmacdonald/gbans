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
import { TableHeadingCell } from './TableHeadingCell.tsx';

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
            header: () => <TableHeadingCell name={'Weapon'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Link
                        component={RouterLink}
                        to={'/stats/weapon/$weapon_id'}
                        params={{ weapon_id: stats?.data[info.row.index].weapon_id }}
                    >
                        {info.getValue()}
                    </Link>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('kills', {
            header: () => <TableHeadingCell name={'Kills'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('kills_pct', {
            header: () => <TableHeadingCell name={'Kills%'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), defaultFloatFmtPct)}</Typography>
                </TableCellSmall>
            )
        }),

        columnHelper.accessor('shots', {
            header: () => <TableHeadingCell name={'Shots'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('shots_pct', {
            header: () => <TableHeadingCell name={'Shot%'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), defaultFloatFmtPct)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('hits', {
            header: () => <TableHeadingCell name={'Hits'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), defaultFloatFmt)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('hits_pct', {
            header: () => <TableHeadingCell name={'Hits%'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), defaultFloatFmtPct)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('accuracy', {
            header: () => <TableHeadingCell name={'Acc%'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>
                        <Typography>{FmtWhenGt(info.getValue(), defaultFloatFmtPct)}</Typography>
                    </Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('airshots', {
            header: () => <TableHeadingCell name={'As'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('airshots_pct', {
            header: () => <TableHeadingCell name={'As%'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), defaultFloatFmtPct)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('backstabs', {
            header: () => <TableHeadingCell name={'Bs'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('backstabs_pct', {
            header: () => <TableHeadingCell name={'Bs%'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), defaultFloatFmtPct)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('headshots', {
            header: () => <TableHeadingCell name={'Hs'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('headshots_pct', {
            header: () => <TableHeadingCell name={'Hs%'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), defaultFloatFmtPct)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('damage', {
            header: () => <TableHeadingCell name={'Dmg'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('damage_pct', {
            header: () => <TableHeadingCell name={'Dmg%'} />,
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
