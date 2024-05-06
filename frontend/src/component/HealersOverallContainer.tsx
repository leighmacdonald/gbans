import { useState } from 'react';
import HealthAndSafetyIcon from '@mui/icons-material/HealthAndSafety';
import Typography from '@mui/material/Typography';
import { useQuery } from '@tanstack/react-query';
import { createColumnHelper, getCoreRowModel, getPaginationRowModel, useReactTable } from '@tanstack/react-table';
import { apiGetHealersOverall, HealingOverallResult } from '../api';
import { LazyResult, RowsPerPage } from '../util/table.ts';
import { defaultFloatFmt, defaultFloatFmtPct, humanCount } from '../util/text.tsx';
import { ContainerWithHeader } from './ContainerWithHeader';
import { DataTable } from './DataTable.tsx';
import FmtWhenGt from './FmtWhenGT.tsx';
import { PaginatorLocal } from './PaginatorLocal.tsx';
import { PersonCell } from './PersonCell';
import { TableCellSmall } from './TableCellSmall.tsx';
import { TableHeadingCell } from './TableHeadingCell.tsx';

export const HealersOverallContainer = () => {
    const { data, isLoading } = useQuery({
        queryKey: ['statsHealingOverall'],
        queryFn: async () => {
            return await apiGetHealersOverall();
        }
    });

    return (
        <ContainerWithHeader title={'Top 250 Medic By Healing'} iconLeft={<HealthAndSafetyIcon />}>
            <StatsHealingOverall stats={data ?? { data: [], count: 0 }} isLoading={isLoading} />
        </ContainerWithHeader>
    );
};

const columnHelper = createColumnHelper<HealingOverallResult>();

const StatsHealingOverall = ({ stats, isLoading }: { stats: LazyResult<HealingOverallResult>; isLoading: boolean }) => {
    const [pagination, setPagination] = useState({
        pageIndex: 0, //initial page index
        pageSize: RowsPerPage.TwentyFive //default page size
    });

    const columns = [
        columnHelper.accessor('rank', {
            header: () => <TableHeadingCell name={'#'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{info.getValue()}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('personaname', {
            header: () => <TableHeadingCell name={'Name'} />,
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
        columnHelper.accessor('healing', {
            header: () => <TableHeadingCell name={'Healing'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
                </TableCellSmall>
            )
        }),

        columnHelper.accessor('assists', {
            header: () => <TableHeadingCell name={'A'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('deaths', {
            header: () => <TableHeadingCell name={'D'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('kad', {
            header: () => <TableHeadingCell name={'KAD'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), defaultFloatFmt)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('hpm', {
            header: () => <TableHeadingCell name={'HPM'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), () => defaultFloatFmt(info.getValue()))}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('damage_taken', {
            header: () => <TableHeadingCell name={'DT'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('dtm', {
            header: () => <TableHeadingCell name={'DTM%'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), () => defaultFloatFmtPct(info.getValue()))}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('dominations', {
            header: () => <TableHeadingCell name={'DM'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('drops', {
            header: () => <TableHeadingCell name={'Dr'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('charges_uber', {
            header: () => <TableHeadingCell name={'Ub'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('charges_kritz', {
            header: () => <TableHeadingCell name={'Kr'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('charges_quickfix', {
            header: () => <TableHeadingCell name={'Qf'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('charges_vacc', {
            header: () => <TableHeadingCell name={'Va'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{FmtWhenGt(info.getValue(), humanCount)}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('win_rate', {
            header: () => <TableHeadingCell name={'WR%'} />,
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
