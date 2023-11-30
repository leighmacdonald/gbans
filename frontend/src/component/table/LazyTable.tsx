import React, { ReactNode } from 'react';
import {
    TableFooter,
    TablePagination,
    TableSortLabel,
    Theme
} from '@mui/material';
import Table from '@mui/material/Table';
import TableBody from '@mui/material/TableBody';
import TableCell from '@mui/material/TableCell';
import TableContainer from '@mui/material/TableContainer';
import TableHead from '@mui/material/TableHead';
import TableRow from '@mui/material/TableRow';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { SxProps, useTheme } from '@mui/material/styles';

export enum RowsPerPage {
    Ten = 10,
    TwentyFive = 25,
    Fifty = 50,
    Hundred = 100
}

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
}

export const defaultRenderer = (
    _: unknown,
    value: unknown,
    type: string
): ReactNode => {
    switch (type) {
        case 'boolean':
            return value ? 'Yes' : 'No';
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

export const descendingComparator = <T,>(a: T, b: T, orderBy: keyof T) =>
    b[orderBy] < a[orderBy] ? -1 : b[orderBy] > a[orderBy] ? 1 : 0;

export type Order = 'asc' | 'desc';

export interface LazyTableProps<T> {
    columns: HeadingCell<T>[];
    sortColumn: keyof T;
    onSortColumnChanged: (column: keyof T) => void;
    onSortOrderChanged: (order: Order) => void;
    sortOrder: Order;
    rows: T[];
    loading?: boolean;
    showPager?: boolean;
    hideHeader?: boolean;
    onRowsPerPageChange?: React.ChangeEventHandler<
        HTMLTextAreaElement | HTMLInputElement
    >;
    onPageChange?: (
        event: React.MouseEvent<HTMLButtonElement> | null,
        page: number
    ) => void;
    // Current page, used to calculate db offset
    page?: number;
    // Total rows in query without paging.
    count?: number;
    rowsPerPage?: RowsPerPage;
}

export interface TableBodyRows<T> {
    columns: HeadingCell<T>[];
    rows: T[];
}

export const LazyTableBody = <T,>({ rows, columns }: TableBodyRows<T>) => {
    return (
        <TableBody>
            {rows.map((row, idx) => {
                return (
                    <TableRow key={`row-${idx}`} hover>
                        {columns.map((col: HeadingCell<T>, colIdx) => {
                            const value = (col?.renderer ?? defaultRenderer)(
                                row,
                                row[col.sortKey as keyof T],
                                col?.sortType ?? 'string'
                            );
                            const style = {
                                paddingLeft: 0.5,
                                paddingRight: 0.5,
                                paddingTop: 0,
                                paddingBottom: 0,
                                width: col?.width ?? 'auto',
                                ...(col?.style ? col.style(row) : {})
                            };
                            return (
                                <TableCell
                                    variant="body"
                                    key={`col-${colIdx}`}
                                    align={col?.align ?? 'right'}
                                    padding={'none'}
                                    onClick={() => {
                                        if (col.onClick) {
                                            col.onClick(row);
                                        }
                                    }}
                                    sx={style}
                                >
                                    {value}
                                </TableCell>
                            );
                        })}
                    </TableRow>
                );
            })}
        </TableBody>
    );
};

export interface LazyTableHeaderProps<T> {
    columns: HeadingCell<T>[];
    bgColor: string;
    onSortColumnChanged: (column: keyof T) => void;
    onSortOrderChanged: (order: Order) => void;
    sortColumn: keyof T;
    order: Order;
}

export const LazyTableHeader = <T,>({
    columns,
    bgColor,
    sortColumn,
    order,
    onSortColumnChanged,
    onSortOrderChanged
}: LazyTableHeaderProps<T>) => {
    return (
        <TableHead>
            <TableRow>
                {columns.map((col) => {
                    return (
                        <TableCell
                            variant="head"
                            align={col.align ?? 'right'}
                            key={`${col.tooltip}-${col.label}-${
                                col.sortKey as string
                            }-${col.virtualKey}`}
                            sortDirection={order}
                            padding={'none'}
                            sx={{
                                width: col?.width ?? 'auto',
                                backgroundColor: bgColor,
                                padding: 0.5
                            }}
                        >
                            {col.sortable ? (
                                <TableSortLabel
                                    title={col.tooltip}
                                    active={sortColumn === col.sortKey}
                                    direction={order}
                                    onClick={() => {
                                        if (col.sortKey) {
                                            if (sortColumn == col.sortKey) {
                                                onSortOrderChanged(
                                                    order == 'asc'
                                                        ? 'desc'
                                                        : 'asc'
                                                );
                                            } else {
                                                onSortColumnChanged(
                                                    col.sortKey
                                                );
                                            }
                                        }
                                    }}
                                >
                                    <Typography
                                        padding={0}
                                        sx={{
                                            fontWeight: 'bold'
                                        }}
                                        variant={'button'}
                                    >
                                        {col.label}
                                    </Typography>
                                </TableSortLabel>
                            ) : (
                                <Tooltip title={col.tooltip}>
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
                                        variant={'button'}
                                    >
                                        {col.label}
                                    </Typography>
                                </Tooltip>
                            )}
                        </TableCell>
                    );
                })}
            </TableRow>
        </TableHead>
    );
};

export const LazyTable = <T,>({
    columns,
    sortOrder,
    sortColumn,
    rows,
    onSortColumnChanged,
    onSortOrderChanged,
    onRowsPerPageChange,
    onPageChange,
    showPager,
    page,
    count,
    rowsPerPage,
    loading,
    hideHeader
}: LazyTableProps<T>) => {
    const theme = useTheme();

    return (
        <TableContainer>
            <Table size={'small'}>
                {!hideHeader && (
                    <LazyTableHeader<T>
                        columns={columns}
                        sortColumn={sortColumn}
                        onSortColumnChanged={onSortColumnChanged}
                        order={sortOrder}
                        bgColor={theme.palette.background.paper}
                        onSortOrderChanged={onSortOrderChanged}
                    />
                )}
                <LazyTableBody rows={rows} columns={columns} />
                {showPager &&
                page != undefined &&
                count != undefined &&
                rowsPerPage != undefined &&
                onPageChange != undefined ? (
                    <TableFooter>
                        <TableRow>
                            <TablePagination
                                SelectProps={{
                                    disabled:
                                        loading != undefined ? loading : false
                                }}
                                backIconButtonProps={
                                    (loading != undefined ? loading : false)
                                        ? {
                                              disabled: loading
                                          }
                                        : undefined
                                }
                                nextIconButtonProps={
                                    (loading != undefined ? loading : false)
                                        ? {
                                              disabled: loading
                                          }
                                        : undefined
                                }
                                page={page}
                                count={count}
                                showFirstButton
                                showLastButton
                                rowsPerPage={rowsPerPage}
                                onRowsPerPageChange={onRowsPerPageChange}
                                onPageChange={onPageChange}
                            />
                        </TableRow>
                    </TableFooter>
                ) : (
                    <></>
                )}
            </Table>
        </TableContainer>
    );
};
