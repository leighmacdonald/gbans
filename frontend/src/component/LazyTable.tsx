import { defaultRenderer, HeadingCell, Order } from './DataTable';
import React from 'react';
import TableContainer from '@mui/material/TableContainer';
import Table from '@mui/material/Table';
import useTheme from '@mui/material/styles/useTheme';
import TableBody from '@mui/material/TableBody';
import TableRow from '@mui/material/TableRow';
import TableCell from '@mui/material/TableCell';
import TableHead from '@mui/material/TableHead';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';

export interface LazyTableProps<T> {
    columns: HeadingCell<T>[];
    sortColumn: keyof T;
    onSortColumnChanged: (column: keyof T) => void;
    onSortOrderChanged: (order: Order) => void;
    sortOrder: Order;
    rows: T[];
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
                    <TableRow key={`row-${idx}`}>
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
                                    onSortOrderChanged(
                                        order === 'asc' ? 'desc' : 'asc'
                                    );
                                } else {
                                    onSortColumnChanged(col.sortKey as keyof T);
                                    onSortOrderChanged('desc');
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

export const LazyTable = <T,>({
    columns,
    sortOrder,
    sortColumn,
    rows,
    onSortColumnChanged,
    onSortOrderChanged
}: LazyTableProps<T>) => {
    const theme = useTheme();
    // eslint-disable-next-line @typescript-eslint/no-unused-vars

    return (
        <TableContainer>
            <Table>
                <LazyTableHeader<T>
                    columns={columns}
                    sortColumn={sortColumn}
                    onSortColumnChanged={onSortColumnChanged}
                    order={sortOrder}
                    bgColor={theme.palette.background.paper}
                    onSortOrderChanged={onSortOrderChanged}
                />
                <LazyTableBody rows={rows} columns={columns} />
            </Table>
        </TableContainer>
    );
};
