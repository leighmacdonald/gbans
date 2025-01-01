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
import { renderDateTime } from '../util/time.ts';
import { DataTable } from './DataTable.tsx';
import { TableCellSmall } from './TableCellSmall.tsx';

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
            header: 'Created',
            size: 120,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{renderDateTime(info.getValue())}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('persona_name', {
            header: 'Name',
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{info.getValue()}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('ip_addr', {
            header: 'IP Address',
            size: 120,
            cell: (info) => (
                <TableCellSmall>
                    <Typography>{info.getValue()}</Typography>
                </TableCellSmall>
            )
        }),
        columnHelper.accessor('server_id', {
            header: 'Server',
            size: 120,
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
