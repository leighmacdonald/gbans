import { useState } from 'react';
import {
    ColumnDef,
    ColumnFiltersState,
    getCoreRowModel,
    getFilteredRowModel,
    getPaginationRowModel,
    getSortedRowModel,
    SortingState,
    useReactTable
} from '@tanstack/react-table';
import { RowsPerPage } from '../util/table.ts';
import { DataTable } from './DataTable.tsx';
import { PaginatorLocal } from './PaginatorLocal.tsx';

type FullTableProps<T> = {
    data: T[];
    isLoading: boolean;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    columns: ColumnDef<T, any>[];
    enableSorting?: boolean;
    enableFiltering?: boolean;
    enablePaging?: boolean;
    pageSize?: RowsPerPage;
};

// Higher level table component. Most/all tables with client side data should use this eventually.
export const FullTable = <T,>({
    data,
    columns,
    isLoading,
    enableSorting = true,
    enablePaging = true,
    enableFiltering = true,
    pageSize = RowsPerPage.TwentyFive
}: FullTableProps<T>) => {
    const [pagination, setPagination] = useState({
        pageIndex: 0, //initial page index
        pageSize: pageSize //default page size
    });
    const [sorting, setSorting] = useState<SortingState>([]);
    const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>([]);

    const table = useReactTable<T>({
        data: data,
        columns: columns,
        autoResetPageIndex: true,
        getCoreRowModel: getCoreRowModel(),
        getFilteredRowModel: enableFiltering ? getFilteredRowModel() : undefined,
        getPaginationRowModel: enablePaging ? getPaginationRowModel() : undefined,
        getSortedRowModel: enableSorting ? getSortedRowModel() : undefined,
        onPaginationChange: setPagination,
        onSortingChange: setSorting,
        onColumnFiltersChange: setColumnFilters,
        state: {
            sorting,
            pagination,
            columnFilters
        }
    });

    return (
        <>
            <DataTable table={table} isLoading={isLoading} />
            {enablePaging && (
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
                    count={table.getRowCount()}
                    rows={pagination.pageSize}
                    page={pagination.pageIndex}
                />
            )}
        </>
    );
};
