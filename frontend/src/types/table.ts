import {
    ColumnFilter,
    ColumnFiltersState,
    ColumnSort,
    OnChangeFn,
    PaginationState,
    SortingState
} from '@tanstack/react-table';
import { RowsPerPage } from '../util/table.ts';

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

export const initSortOrder = (id?: string, desc?: 'desc' | 'asc'): ColumnSort[] => {
    return id ? [{ id: id, desc: (desc ?? 'desc') == desc }] : [];
};

export const initColumnFilter = (id?: string, value?: string): ColumnFilter[] => {
    return id && value ? [{ id: id, value: value }] : [];
};

export const initPagination = (pageIndex?: number, pageSize?: number): PaginationState => {
    return { pageIndex: pageIndex ?? 0, pageSize: pageSize ?? RowsPerPage.TwentyFive };
};
