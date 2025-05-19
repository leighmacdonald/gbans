import { ColumnFilter, ColumnSort, PaginationState } from '@tanstack/react-table';
import { intervalToDuration } from 'date-fns';
import { z } from 'zod';
import { DataCount } from '../api';
import { emptyOrNullString } from './types.ts';

export enum RowsPerPage {
    Ten = 10,
    TwentyFive = 25,
    Fifty = 50,
    Hundred = 100
}

export const isPermanentBan = (start: Date, end: Date): boolean => {
    const dur = intervalToDuration({
        start,
        end
    });
    const { years } = dur;
    return years != null && years > 5;
};

export const commonTableSearchSchema = {
    pageIndex: z.number().optional().catch(0),
    pageSize: z.number().optional().catch(RowsPerPage.TwentyFive),
    sortOrder: z.enum(['desc', 'asc']).optional().catch('desc')
};

export const makeCommonTableSearchSchema = (sortColumns: readonly [string, ...string[]]) => {
    return { ...commonTableSearchSchema, sortColumn: z.enum(sortColumns).optional() };
};

export type Order = 'asc' | 'desc';

export interface LazyResult<T> extends DataCount {
    data: T[];
}

export const initSortOrder = (
    id: string | undefined,
    desc: 'desc' | 'asc' | undefined,
    def: ColumnSort
): ColumnSort[] => {
    return id ? [{ id: id, desc: (desc ?? 'desc') == desc }] : [def];
};

export const initColumnFilter = (filters: Record<string, unknown>): ColumnFilter[] => {
    return Object.keys(filters)
        .filter(
            (k) =>
                filters[k] != undefined ||
                commonPropNames.includes(k) ||
                (typeof filters[k] == 'string' && !emptyOrNullString(filters[k] as string)) ||
                (typeof filters[k] == 'number' && Number(filters[k]) > 0)
        )
        .map((k) => {
            return { id: k, value: filters[k] };
        });
};

export const initPagination = (pageIndex?: number, pageSize?: number): PaginationState => {
    return { pageIndex: pageIndex ?? 0, pageSize: pageSize ?? RowsPerPage.TwentyFive };
};

const commonPropNames = ['page', 'rows', 'sortOrder', 'sortDesc', 'pageIndex', 'pageSize'];
