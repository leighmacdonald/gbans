import Table from '@mui/material/Table';
import Paper from '@mui/material/Paper';
import TableContainer from '@mui/material/TableContainer';
import React, { ReactNode, useMemo, useState } from 'react';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import TableCell from '@mui/material/TableCell';
import TableBody from '@mui/material/TableBody';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import useTheme from '@mui/material/styles/useTheme';

const stableSort = <T,>(array: T[], comparator: (a: T, b: T) => number) => {
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

export interface HeadingCell<T> {
    label: string;
    align?: 'inherit' | 'left' | 'center' | 'right' | 'justify';
    tooltip: string;
    sortKey?: keyof T;
    width?: number | string;
    sortType?: 'number' | 'string' | 'date' | 'float';
    virtual?: boolean;
    virtualKey?: string;
    sortable?: boolean;
    // Custom cell render function for complex types
    renderer?: (obj: T, value: unknown, type: string) => ReactNode;
}

export interface UserTableProps<T> {
    columns: HeadingCell<T>[];
    defaultSortColumn: keyof T;
    rowsPerPage: number;
    rows: T[];
}

export function defaultRenderer(
    _: unknown,
    value: unknown,
    type: string
): ReactNode {
    switch (type) {
        case 'date':
            return new Date(value as string).toDateString();
        case 'float':
            return ((value as number) ?? 0).toFixed(2);
        default:
            return `${value}`;
    }
}

const descendingComparator = <T,>(a: T, b: T, orderBy: keyof T) => {
    if (b[orderBy] < a[orderBy]) {
        return -1;
    }
    if (b[orderBy] > a[orderBy]) {
        return 1;
    }
    return 0;
};

export type Order = 'asc' | 'desc';

export const UserTable = <T,>({
    columns,
    rows,
    defaultSortColumn,
    rowsPerPage
}: UserTableProps<T>) => {
    const theme = useTheme();
    const [page] = React.useState(0);
    const [order, setOrder] = React.useState<Order>('desc');
    const [sortColumn, setSortColumn] = useState<keyof T>(defaultSortColumn);

    const compare = useMemo(() => {
        return (order: Order, orderBy: keyof T): ((a: T, b: T) => number) =>
            order === 'desc'
                ? (a, b) => descendingComparator(a, b, orderBy)
                : (a, b) => -descendingComparator(a, b, orderBy);
    }, []);

    const sorted = useMemo(() => {
        return stableSort<T>(rows, compare(order, sortColumn)).slice(
            page * rowsPerPage,
            page * rowsPerPage + rowsPerPage
        );
    }, [rows, compare, order, sortColumn, page, rowsPerPage]);

    // const getColumn = (sortKey: keyof T): HeadingCell<T> | undefined =>
    //     first(columns.filter((c) => c.sortKey == sortKey));

    return (
        <TableContainer component={Paper}>
            <Table>
                <TableHead>
                    <TableRow>
                        {columns.map((col) => {
                            return (
                                <TableCell
                                    align={col.align ?? 'right'}
                                    key={col.label}
                                    sx={{
                                        width: col?.width ?? 'auto',
                                        '&:hover': {
                                            cursor: 'pointer'
                                        }
                                    }}
                                    onClick={() => {
                                        if (!col.sortable || col.virtual) {
                                            return;
                                        }
                                        if (col.sortKey === sortColumn) {
                                            setOrder(
                                                order === 'asc' ? 'desc' : 'asc'
                                            );
                                        } else {
                                            setSortColumn(col.sortKey as never);
                                            setOrder('desc');
                                        }
                                    }}
                                >
                                    <Tooltip
                                        title={col.tooltip}
                                        placement={'top'}
                                    >
                                        <Typography variant={'h6'}>
                                            {col.label}
                                        </Typography>
                                    </Tooltip>
                                </TableCell>
                            );
                        })}
                    </TableRow>
                </TableHead>
                <TableBody>
                    {sorted.map((row, rowIdx) => {
                        return (
                            <TableRow
                                key={rowIdx}
                                sx={{
                                    '&:hover': {
                                        backgroundColor:
                                            theme.palette.background.default
                                    }
                                }}
                            >
                                {columns.map((col, colIdx) => {
                                    const value = (
                                        col?.renderer ?? defaultRenderer
                                    )(
                                        row,
                                        // eslint-disable-next-line @typescript-eslint/no-explicit-any
                                        (row as any)[col.sortKey],
                                        col?.sortType || 'string'
                                    );
                                    return (
                                        <TableCell
                                            key={`col-${colIdx}`}
                                            align={col?.align ?? 'right'}
                                            sx={{
                                                width: col?.width ?? 'auto',
                                                '&:hover': {
                                                    cursor: 'pointer'
                                                }
                                            }}
                                        >
                                            <Typography variant={'body1'}>
                                                {value}
                                            </Typography>
                                        </TableCell>
                                    );
                                })}
                            </TableRow>
                        );
                    })}
                </TableBody>
            </Table>
        </TableContainer>
    );
};
