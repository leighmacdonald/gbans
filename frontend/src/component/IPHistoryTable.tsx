import Typography from '@mui/material/Typography';
import {
    createColumnHelper,
    getCoreRowModel,
    getPaginationRowModel,
    OnChangeFn,
    PaginationState,
    TableOptions,
    useReactTable
} from '@tanstack/react-table';
import { PersonConnection } from '../api';
import { LazyResult } from '../util/table.ts';
import { renderDateTime } from '../util/text.tsx';
import { DataTable } from './DataTable.tsx';
import { TableCellSmall } from './TableCellSmall.tsx';
import { TableHeadingCell } from './TableHeadingCell.tsx';

const columnHelper = createColumnHelper<PersonConnection>();

export const IPHistoryTable = ({
    connections,
    isLoading,
    manualPaging = true,
    pagination,
    setPagination
}: {
    connections: LazyResult<PersonConnection>;
    isLoading: boolean;
    manualPaging?: boolean;
    pagination?: PaginationState;
    setPagination?: OnChangeFn<PaginationState>;
}) => {
    const columns = [
        columnHelper.accessor('created_on', {
            header: () => <TableHeadingCell name={'Created'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{renderDateTime(info.getValue())}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('persona_name', {
            header: () => <TableHeadingCell name={'Name'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{info.getValue()}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('ip_addr', {
            header: () => <TableHeadingCell name={'IP Address'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{info.getValue()}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('server_id', {
            header: () => <TableHeadingCell name={'Server'} />,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{connections.data[info.row.index].server_name_short}</Typography>
                </TableCellSmall>
            )
        })
    ];

    const opts: TableOptions<PersonConnection> = {
        data: connections.data,
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        manualPagination: manualPaging,
        autoResetPageIndex: true,
        ...(manualPaging
            ? {}
            : {
                  manualPagination: false,
                  onPaginationChange: setPagination,
                  getPaginationRowModel: getPaginationRowModel(),
                  state: { pagination }
              })
    };

    const table = useReactTable(opts);

    return <DataTable table={table} isLoading={isLoading} />;
};
