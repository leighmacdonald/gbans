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

export const initSortOrder = (
    id: string | undefined,
    desc: 'desc' | 'asc' | undefined,
    def: ColumnSort
): ColumnSort[] => {
    return id ? [{ id: id, desc: (desc ?? 'desc') == desc }] : [def];
};

export const initColumnFilter = (filters: Record<string, unknown>): ColumnFilter[] => {
    return Object.keys(filters)
        .filter((k) => filters[k] != undefined)
        .map((k) => {
            return { id: k, value: filters[k] };
        });
};

export const initPagination = (pageIndex?: number, pageSize?: number): PaginationState => {
    return { pageIndex: pageIndex ?? 0, pageSize: pageSize ?? RowsPerPage.TwentyFive };
};
