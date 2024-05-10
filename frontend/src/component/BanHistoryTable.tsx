import { useState } from 'react';
import Typography from '@mui/material/Typography';
import { useNavigate } from '@tanstack/react-router';
import {
    createColumnHelper,
    getCoreRowModel,
    getPaginationRowModel,
    TableOptions,
    useReactTable
} from '@tanstack/react-table';
import { BanReasons, SteamBanRecord } from '../api';
import { LazyResult, RowsPerPage } from '../util/table.ts';
import { renderDateTime } from '../util/text.tsx';
import { DataTable } from './DataTable.tsx';
import { PersonCell } from './PersonCell.tsx';
import { TableCellBool } from './TableCellBool.tsx';
import { TableCellSmall } from './TableCellSmall.tsx';
import { TableHeadingCell } from './TableHeadingCell.tsx';

const columnHelper = createColumnHelper<SteamBanRecord>();

export const BanHistoryTable = ({ bans, isLoading }: { bans: LazyResult<SteamBanRecord>; isLoading: boolean }) => {
    const navigate = useNavigate();
    const [pagination, setPagination] = useState({
        pageIndex: 0, //initial page index
        pageSize: RowsPerPage.TwentyFive //default page size
    });

    const columns = [
        columnHelper.accessor('deleted', {
            header: () => <TableHeadingCell name={'A'} />,
            cell: (info) => {
                return <TableCellBool enabled={!info.getValue()} />;
            }
        }),
        columnHelper.accessor('created_on', {
            header: () => <TableHeadingCell name={'Created'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography align={'center'}>{renderDateTime(info.getValue())}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('source_id', {
            header: () => <TableHeadingCell name={'Author'} />,
            cell: (info) => (
                <PersonCell
                    steam_id={bans.data[info.row.index].source_id}
                    avatar_hash={bans.data[info.row.index].source_avatarhash}
                    personaname={bans.data[info.row.index].source_personaname}
                    onClick={async () => {
                        await navigate({
                            params: { steamId: bans.data[info.row.index].source_id },
                            to: `/profile/$steamId`
                        });
                    }}
                />
            )
        }),
        columnHelper.accessor('reason', {
            header: () => <TableHeadingCell name={'Reason'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography padding={0} variant={'body1'}>
                        {BanReasons[info.getValue()]}
                    </Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('reason_text', {
            header: () => <TableHeadingCell name={'Custom'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography padding={0} variant={'body1'}>
                        {info.getValue()}
                    </Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('unban_reason_text', {
            header: () => <TableHeadingCell name={'Unban Reason'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography padding={0} variant={'body1'}>
                        {info.getValue()}
                    </Typography>
                </TableCellSmall>
            )
        })
    ];

    const opts: TableOptions<SteamBanRecord> = {
        data: bans.data,
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        manualPagination: false,
        autoResetPageIndex: true,
        onPaginationChange: setPagination,
        getPaginationRowModel: getPaginationRowModel(),
        state: { pagination }
    };

    const table = useReactTable(opts);

    return <DataTable table={table} isLoading={isLoading} />;
};
