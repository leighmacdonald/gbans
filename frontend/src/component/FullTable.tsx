import { ToOptions, useNavigate } from '@tanstack/react-router';
import {
    ColumnDef,
    ColumnFiltersState,
    getCoreRowModel,
    getFilteredRowModel,
    getPaginationRowModel,
    getSortedRowModel,
    OnChangeFn,
    PaginationState,
    SortingState,
    useReactTable
} from '@tanstack/react-table';
import { DataTable } from './DataTable.tsx';
import { PaginatorLocal } from './PaginatorLocal.tsx';

type FullTableProps<T> = {
    data: T[];
    isLoading: boolean;
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    columns: ColumnDef<T, any>[];
    columnFilters?: ColumnFiltersState;
    pagination: PaginationState;
    setPagination: OnChangeFn<PaginationState>;
    sorting?: SortingState;
    infinitePage?: boolean;
    toOptions: ToOptions;
};

// Higher level table component. Most/all tables with client side data should use this eventually.
export const FullTable = <T,>({
    data,
    columns,
    isLoading,
    pagination,
    setPagination,
    infinitePage = false,
    sorting = undefined,
    columnFilters = undefined,
    toOptions
}: FullTableProps<T>) => {
    const navigate = useNavigate();

    const table = useReactTable<T>({
        data: data,
        columns: columns,
        autoResetPageIndex: true,
        getCoreRowModel: getCoreRowModel(),
        getFilteredRowModel: columnFilters ? getFilteredRowModel() : undefined,
        getPaginationRowModel: pagination ? getPaginationRowModel() : undefined,
        getSortedRowModel: sorting ? getSortedRowModel() : undefined,
        state: {
            sorting,
            pagination,
            columnFilters
        }
    });

    return (
        <>
            <DataTable table={table} isLoading={isLoading} />
            {pagination && setPagination && (
                <PaginatorLocal
                    onRowsChange={async (rows) => {
                        setPagination((prev) => {
                            return { ...prev, pageSize: rows };
                        });
                        await navigate({ ...toOptions, search: (search) => ({ ...search, pageSize: rows }) });
                    }}
                    onPageChange={async (page) => {
                        setPagination((prev) => {
                            return { ...prev, pageIndex: page };
                        });
                        await navigate({ ...toOptions, search: (search) => ({ ...search, pageIndex: page }) });
                    }}
                    count={infinitePage ? -1 : table.getRowCount()}
                    rows={pagination.pageSize}
                    page={pagination.pageIndex}
                />
            )}
        </>
    );
};
