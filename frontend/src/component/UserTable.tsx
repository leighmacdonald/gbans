import Table from '@mui/material/Table';
import TableContainer from '@mui/material/TableContainer';
import React, { ReactNode, useEffect, useMemo, useState } from 'react';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import TableCell from '@mui/material/TableCell';
import TableBody from '@mui/material/TableBody';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import useTheme from '@mui/material/styles/useTheme';
import { Pagination, Select } from '@mui/material';
import TextField from '@mui/material/TextField';
import Stack from '@mui/material/Stack';
import MenuItem from '@mui/material/MenuItem';

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
    queryValue?: (obj: T) => string;
}

export interface UserTableProps<T> {
    columns: HeadingCell<T>[];
    defaultSortColumn: keyof T;
    rowsPerPage: number;
    rows: T[];
    onRowClick?: (value: T) => void;
    query?: string;
}

export const defaultRenderer = (
    _: unknown,
    value: unknown,
    type: string
): ReactNode => {
    switch (type) {
        case 'date':
            return new Date(value as string).toDateString();
        case 'float':
            return (
                <Typography variant={'body1'}>
                    {((value as number) ?? 0).toFixed(2)}
                </Typography>
            );
        default:
            return <Typography variant={'body1'}>{value as string}</Typography>;
    }
};

const descendingComparator = <T,>(a: T, b: T, orderBy: keyof T) =>
    b[orderBy] < a[orderBy] ? -1 : b[orderBy] > a[orderBy] ? 1 : 0;

export type Order = 'asc' | 'desc';

export const UserTable = <T,>({
    columns,
    rows,
    defaultSortColumn,
    rowsPerPage,
    onRowClick
}: UserTableProps<T>) => {
    const theme = useTheme();
    const [page, setPage] = useState(0);
    const [order, setOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] = useState<keyof T>(defaultSortColumn);
    const [pageCount, setPageCount] = useState(0);
    const [rowPerPageCount, setRowPerPageCount] = useState(rowsPerPage ?? 25);
    const [query, setQuery] = useState('');

    const sorted = useMemo(() => {
        const compare = (
            order: Order,
            orderBy: keyof T
        ): ((a: T, b: T) => number) =>
            order === 'desc'
                ? (a, b) => descendingComparator(a, b, orderBy)
                : (a, b) => -descendingComparator(a, b, orderBy);
        const filterText = (obj: T): boolean => {
            if (!query) {
                return true;
            }
            return (
                columns.filter(
                    (column) =>
                        column.queryValue instanceof Function &&
                        (column.queryValue(obj) || '')
                            .toLowerCase()
                            .includes(query.toLowerCase())
                ).length > 0
            );
        };
        return stableSort<T>(
            (rows ?? []).filter(filterText),
            compare(order, sortColumn)
        );
    }, [rows, order, sortColumn, query, columns]);

    useEffect(() => {
        setPageCount(Math.ceil(sorted.length / rowPerPageCount));
    }, [rowPerPageCount, sorted]);

    return (
        <TableContainer>
            <Stack direction={'row'} justifyContent="space-between" padding={1}>
                <TextField
                    sx={{ padding: 0 }}
                    // label={'Filter'}
                    value={query}
                    placeholder={'Filter'}
                    variant={'standard'}
                    onChange={(event) => {
                        setQuery(event.target.value);
                    }}
                />
                <Select<number>
                    sx={{ padding: 0 }}
                    variant={'standard'}
                    value={rowPerPageCount}
                    onChange={(event) => {
                        setRowPerPageCount(event.target.value as number);
                    }}
                >
                    {[10, 25, 50, 100].map((v) => (
                        <MenuItem value={v} key={v}>
                            {v}
                        </MenuItem>
                    ))}
                </Select>
                <Pagination
                    variant={'text'}
                    count={pageCount}
                    showFirstButton
                    showLastButton
                    onChange={(_, newPage) => {
                        setPage(newPage - 1);
                    }}
                />
            </Stack>

            <Table size={'small'} padding={'normal'} stickyHeader={true}>
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
                                        },
                                        backgroundColor:
                                            theme.palette.background.paper
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
                                        <Typography
                                            padding={0}
                                            sx={{
                                                textDecoration:
                                                    col.sortKey != sortColumn
                                                        ? 'none'
                                                        : order == 'asc'
                                                        ? 'underline'
                                                        : 'overline'
                                            }}
                                            variant={'subtitle1'}
                                        >
                                            {col.label}
                                        </Typography>
                                    </Tooltip>
                                </TableCell>
                            );
                        })}
                    </TableRow>
                </TableHead>
                <TableBody>
                    {sorted
                        .slice(
                            page * rowPerPageCount,
                            page * rowPerPageCount + rowPerPageCount
                        )
                        .map((row, rowIdx) => {
                            return (
                                <TableRow
                                    onClick={() => {
                                        onRowClick && onRowClick(row);
                                    }}
                                    key={rowIdx}
                                    sx={{
                                        '&:hover': {
                                            backgroundColor:
                                                theme.palette.background.paper
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
                                                {value}
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
