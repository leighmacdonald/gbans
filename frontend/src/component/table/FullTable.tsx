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
    useReactTable,
    VisibilityState
} from '@tanstack/react-table';
import { LoadingPlaceholder } from '../LoadingPlaceholder.tsx';
import { PaginatorLocal } from '../forum/PaginatorLocal.tsx';
import { DataTable } from './DataTable.tsx';

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
    columnVisibility?: VisibilityState;
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
    toOptions,
    columnVisibility
}: FullTableProps<T>) => {
    const navigate = useNavigate();

    const table = useReactTable<T>({
        data: data,
        enableHiding: true,
        columns: columns,
        autoResetPageIndex: true,
        getCoreRowModel: getCoreRowModel(),
        getFilteredRowModel: columnFilters ? getFilteredRowModel() : undefined,
        getPaginationRowModel: pagination ? getPaginationRowModel() : undefined,
        getSortedRowModel: sorting ? getSortedRowModel() : undefined,
        state: {
            sorting,
            pagination,
            columnFilters,
            columnVisibility
        }
    });

    return (
        <>
            {isLoading ? <LoadingPlaceholder /> : <DataTable table={table} isLoading={isLoading} />}
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
                    count={infinitePage ? -1 : table.getRowCount() || 0}
                    rows={pagination.pageSize}
                    page={pagination.pageIndex}
                />
            )}
        </>
    );
};
