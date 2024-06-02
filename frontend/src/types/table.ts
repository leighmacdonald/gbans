import { ColumnFiltersState, OnChangeFn, PaginationState, SortingState } from '@tanstack/react-table';

export type TablePagination = {
    pagination: PaginationState;
    setPagination: OnChangeFn<PaginationState>;
};

export type TableFilters = {
    columnFilters: ColumnFiltersState;
    setColumnFilters: OnChangeFn<ColumnFiltersState>;
};

export type TableSorting = {
    sorting: SortingState;
    setSorting: OnChangeFn<SortingState>;
};

export type TablePropsAll = TablePagination & TableFilters & TableSorting;
