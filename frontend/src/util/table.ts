import { ReactNode } from 'react';
import { Theme } from '@mui/material';
import { SxProps } from '@mui/material/styles';
import { z } from 'zod';
import { DataCount } from '../api';

export enum RowsPerPage {
    Ten = 10,
    TwentyFive = 25,
    Fifty = 50,
    Hundred = 100
}

export const commonTableSearchSchema = {
    page: z.number().catch(0),
    rows: z.number().catch(RowsPerPage.TwentyFive),
    sortOrder: z.enum(['desc', 'asc']).catch('desc')
};

export const descendingComparator = <T>(a: T, b: T, orderBy: keyof T) => (b[orderBy] < a[orderBy] ? -1 : b[orderBy] > a[orderBy] ? 1 : 0);

export type Order = 'asc' | 'desc';

export interface HeadingCell<T> {
    label: string;
    align?: 'inherit' | 'left' | 'center' | 'right' | 'justify';
    tooltip: string;
    sortKey?: keyof T;
    width?: number | string;
    sortType?: 'number' | 'string' | 'date' | 'float' | 'boolean';
    virtual?: boolean;
    virtualKey?: string;
    sortable?: boolean;
    // Custom cell render function for complex types
    renderer?: (obj: T, value: unknown, type: string) => ReactNode;
    style?: (obj: T) => SxProps<Theme> | undefined;
    onClick?: (row: T) => void;
    hideSm?: boolean;
}

export const compare = <T>(order: Order, orderBy: keyof T): ((a: T, b: T) => number) =>
    order === 'desc' ? (a, b) => descendingComparator(a, b, orderBy) : (a, b) => -descendingComparator(a, b, orderBy);

export const stableSort = <T>(array: T[], comparator: (a: T, b: T) => number) => {
    const stabilizedThis = array.map((el, index) => [el, index] as [T, number]);
    stabilizedThis.sort((a, b) => {
        const order = comparator(a[0], b[0]);
        if (order !== 0) {
            return order;
        }
        return a[1] - b[1];
    });
    return stabilizedThis.map((el) => el[0]);
};

export interface LazyFetchOpts<T> {
    column: keyof T;
    order: Order;
    page: number;
}

export interface LazyResult<T> extends DataCount {
    data: T[];
}
