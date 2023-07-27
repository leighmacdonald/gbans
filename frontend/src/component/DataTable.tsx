import React, { ReactNode, useEffect, useMemo, useState } from 'react';
import Table from '@mui/material/Table';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import TableCell, { tableCellClasses } from '@mui/material/TableCell';
import TableBody from '@mui/material/TableBody';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import useTheme from '@mui/material/styles/useTheme';
import TextField from '@mui/material/TextField';
import Stack from '@mui/material/Stack';
import MenuItem from '@mui/material/MenuItem';
import { LoadingSpinner } from './LoadingSpinner';
import Select from '@mui/material/Select';
import Pagination from '@mui/material/Pagination';

export enum RowsPerPage {
    Ten = 10,
    TwentyFive = 25,
    Fifty = 50,
    Hundred = 100
}

export const stableSort = <T,>(
    array: T[],
    comparator: (a: T, b: T) => number
) => {
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
    tooltip: string | (() => string);
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
    defaultSortOrder?: Order;
    rowsPerPage: RowsPerPage;
    rows: T[];
    onRowClick?: (value: T) => void;
    query?: string;
    isLoading?: boolean;
    preSelectIndex?: number;
    filterFn?: (query: string, rows: T[]) => T[];
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

export const createTableHeader = <T,>(
    columns: HeadingCell<T>[],
    bgColor: string,
    setSortColumn: React.Dispatch<React.SetStateAction<keyof T>>,
    sortColumn: keyof T,
    order: Order,
    setOrder: React.Dispatch<React.SetStateAction<Order>>
) => {
    return (
        <TableHead>
            <TableRow>
                {columns.map((col) => {
                    return (
                        <TableCell
                            align={col.align ?? 'right'}
                            key={col.label}
                            sx={{
                                width: col?.width ?? 'auto',
                                '&:hover': col.sortable
                                    ? {
                                          cursor: 'pointer'
                                      }
                                    : { cursor: 'default' },
                                backgroundColor: bgColor
                            }}
                            onClick={() => {
                                if (!col.sortable || col.virtual) {
                                    return;
                                }
                                if (col.sortKey === sortColumn) {
                                    setOrder(order === 'asc' ? 'desc' : 'asc');
                                } else {
                                    setSortColumn(col.sortKey as never);
                                    setOrder('desc');
                                }
                            }}
                        >
                            <Tooltip
                                title={
                                    col.tooltip instanceof Function
                                        ? col.tooltip()
                                        : col.tooltip
                                }
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
                                    variant={'subtitle2'}
                                >
                                    {col.label}
                                </Typography>
                            </Tooltip>
                        </TableCell>
                    );
                })}
            </TableRow>
        </TableHead>
    );
};

const descendingComparator = <T,>(a: T, b: T, orderBy: keyof T) =>
    b[orderBy] < a[orderBy] ? -1 : b[orderBy] > a[orderBy] ? 1 : 0;

export type Order = 'asc' | 'desc';

export const DataTable = <T,>({
    columns,
    rows,
    defaultSortColumn,
    rowsPerPage,
    onRowClick,
    isLoading,
    defaultSortOrder = 'desc',
    preSelectIndex,
    filterFn
}: UserTableProps<T>) => {
    const theme = useTheme();
    const [page, setPage] = useState(0);
    const [order, setOrder] = useState<Order>(defaultSortOrder);
    const [sortColumn, setSortColumn] = useState<keyof T>(defaultSortColumn);
    const [pageCount, setPageCount] = useState(0);
    const [rowPerPageCount, setRowPerPageCount] = useState(
        rowsPerPage ?? RowsPerPage.TwentyFive
    );
    const [query, setQuery] = useState('');

    useEffect(() => {
        if (!preSelectIndex || preSelectIndex <= 0) {
            return;
        }
        const newVal = Math.ceil(preSelectIndex / rowPerPageCount);
        setPage(newVal);
    }, [preSelectIndex, rowPerPageCount]);

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
        if (filterFn instanceof Function) {
            return stableSort<T>(
                filterFn(query, rows) ?? [],
                compare(order, sortColumn)
            );
        } else {
            return stableSort<T>(
                (rows ?? []).filter(filterText),
                compare(order, sortColumn)
            );
        }
    }, [filterFn, query, columns, rows, order, sortColumn]);

    useEffect(() => {
        setPageCount(Math.ceil(sorted.length / rowPerPageCount));
    }, [rowPerPageCount, sorted]);

    const renderedRows = useMemo(() => {
        return sorted
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
                                    theme.palette.background.default
                            }
                        }}
                    >
                        {columns.map((col: HeadingCell<T>, colIdx) => {
                            const value = (col?.renderer ?? defaultRenderer)(
                                row,
                                row[col.sortKey as keyof T],
                                col?.sortType ?? 'string'
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
            });
    }, [
        columns,
        onRowClick,
        page,
        rowPerPageCount,
        sorted,
        theme.palette.background.default
    ]);

    const tableHead = useMemo(() => {
        return (
            <TableHead>
                <TableRow>
                    {columns.map((col) => {
                        return (
                            <TableCell
                                align={col.align ?? 'right'}
                                key={col.label}
                                sx={{
                                    width: col?.width ?? 'auto',
                                    '&:hover': col.sortable
                                        ? {
                                              cursor: 'pointer'
                                          }
                                        : { cursor: 'default' },
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
                                    title={
                                        col.tooltip instanceof Function
                                            ? col.tooltip()
                                            : col.tooltip
                                    }
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
                                        variant={'subtitle2'}
                                    >
                                        {col.label}
                                    </Typography>
                                </Tooltip>
                            </TableCell>
                        );
                    })}
                </TableRow>
            </TableHead>
        );
    }, [columns, order, sortColumn, theme.palette.background.paper]);

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
                <Select<RowsPerPage>
                    sx={{ padding: 0 }}
                    variant={'standard'}
                    value={rowPerPageCount}
                    onChange={(event) => {
                        setRowPerPageCount(event.target.value as RowsPerPage);
                    }}
                >
                    {[
                        RowsPerPage.Ten,
                        RowsPerPage.TwentyFive,
                        RowsPerPage.Fifty,
                        RowsPerPage.Hundred
                    ].map((v) => (
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

            <Table
                size={'small'}
                padding={'normal'}
                stickyHeader={true}
                sx={{
                    [`& .${tableCellClasses.root}`]: {
                        borderBottomColor: theme.palette.text.disabled
                    }
                }}
            >
                {!isLoading && tableHead}
                {isLoading ? (
                    <TableBody>
                        <TableRow>
                            <TableCell colSpan={columns.length}>
                                <LoadingSpinner />
                            </TableCell>
                        </TableRow>
                    </TableBody>
                ) : (
                    <TableBody>{renderedRows}</TableBody>
                )}
            </Table>
        </TableContainer>
    );
};
