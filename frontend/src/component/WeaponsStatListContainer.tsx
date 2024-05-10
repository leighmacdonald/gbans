import { useState } from 'react';
import InsightsIcon from '@mui/icons-material/Insights';
import Link from '@mui/material/Link';
import Typography from '@mui/material/Typography';
import { useQuery } from '@tanstack/react-query';
import { createColumnHelper, getCoreRowModel, getPaginationRowModel, useReactTable } from '@tanstack/react-table';
import { apiGetWeaponsOverall, WeaponsOverallResult } from '../api';
import { LazyResult, RowsPerPage } from '../util/table.ts';
import { defaultFloatFmt, defaultFloatFmtPct, humanCount } from '../util/text.tsx';
import { ContainerWithHeader } from './ContainerWithHeader';
import { DataTable } from './DataTable.tsx';
import FmtWhenGt from './FmtWhenGT.tsx';
import { PaginatorLocal } from './PaginatorLocal.tsx';
import RouterLink from './RouterLink.tsx';
import { TableCellSmall } from './TableCellSmall.tsx';
import { TableHeadingCell } from './TableHeadingCell.tsx';

export const WeaponsStatListContainer = () => {
    const { data, isLoading } = useQuery({
        queryKey: ['statsWeaponsOverall'],
        queryFn: async () => apiGetWeaponsOverall()
    });

    return (
        <ContainerWithHeader title={'Overall Weapon Stats'} iconLeft={<InsightsIcon />}>
            <StatsWeaponsOverall stats={data ?? { data: [], count: 0 }} isLoading={isLoading} />
        </ContainerWithHeader>
    );
};

const columnHelper = createColumnHelper<WeaponsOverallResult>();

const StatsWeaponsOverall = ({ stats, isLoading }: { stats: LazyResult<WeaponsOverallResult>; isLoading: boolean }) => {
    const [pagination, setPagination] = useState({
        pageIndex: 0, //initial page index
        pageSize: RowsPerPage.TwentyFive //default page size
    });

    const columns = [
        columnHelper.accessor('weapon_id', {
            header: () => <TableHeadingCell name={'#'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Link
                        component={RouterLink}
                        to={'/stats/weapon/$weapon_id'}
                        params={{ weapon_id: info.getValue() }}
                    >
                        {stats.data[info.row.index].name}
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
            header: () => <TableHeadingCell name={'Shots%'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), defaultFloatFmt)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('hits', {
            header: () => <TableHeadingCell name={'Hits'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
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
                    <Typography>{FmtWhenGt(info.getValue(), defaultFloatFmtPct)}</Typography>
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
        data: stats.data,
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
                count={stats.count}
                rows={pagination.pageSize}
                page={pagination.pageIndex}
            />
        </>
    );
};
