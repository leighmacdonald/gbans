import {
    createTableHeader,
    defaultRenderer,
    HeadingCell,
    Order,
    RowsPerPage
} from './DataTable';
import React, { useEffect, useMemo, useState } from 'react';
import TableContainer from '@mui/material/TableContainer';
import Table from '@mui/material/Table';
import Pagination from '@mui/material/Pagination';
import useTheme from '@mui/material/styles/useTheme';
import Stack from '@mui/material/Stack';
import TableBody from '@mui/material/TableBody';
import TableRow from '@mui/material/TableRow';
import TableCell from '@mui/material/TableCell';

export interface LazyTableProps<T> {
    columns: HeadingCell<T>[];
    defaultSortColumn: keyof T;
    defaultSortOrder?: Order;
    rowsPerPage: RowsPerPage;
    preSelectIndex?: number;
    rows: T[];
}

export const LazyTable = <T,>({
    columns,
    defaultSortOrder = 'desc',
    preSelectIndex,
    defaultSortColumn,
    rowsPerPage,
    rows
}: LazyTableProps<T>) => {
    const theme = useTheme();
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
    const [_, setPage] = useState(0);
    const [order, setOrder] = useState<Order>(defaultSortOrder);
    const [sortColumn, setSortColumn] = useState<keyof T>(defaultSortColumn);
    const [pageCount] = useState(0);
    const [rowPerPageCount] = useState(rowsPerPage ?? RowsPerPage.TwentyFive);

    useEffect(() => {
        if (!preSelectIndex || preSelectIndex <= 0) {
            return;
        }
        const newVal = Math.ceil(preSelectIndex / rowPerPageCount);
        setPage(newVal);
    }, [preSelectIndex, rowPerPageCount]);

    const tableHead = useMemo(() => {
        return createTableHeader<T>(
            columns,
            theme.palette.background.paper,
            setSortColumn,
            sortColumn,
            order,
            setOrder
        );
    }, [columns, order, sortColumn, theme.palette.background.paper]);

    return (
        <Stack>
            <TableContainer>
                <Table>
                    {tableHead}
                    <TableBody>
                        {rows.map((row, idx) => {
                            return (
                                <TableRow key={`row-${idx}`}>
                                    {columns.map(
                                        (col: HeadingCell<T>, colIdx) => {
                                            const value = (
                                                col?.renderer ?? defaultRenderer
                                            )(
                                                row,
                                                row[col.sortKey as keyof T],
                                                col?.sortType ?? 'string'
                                            );
                                            return (
                                                <TableCell
                                                    key={`col-${colIdx}`}
                                                    align={
                                                        col?.align ?? 'right'
                                                    }
                                                    sx={{
                                                        width:
                                                            col?.width ??
                                                            'auto',
                                                        '&:hover': {
                                                            cursor: 'pointer'
                                                        }
                                                    }}
                                                >
                                                    {value}
                                                </TableCell>
                                            );
                                        }
                                    )}
                                </TableRow>
                            );
                        })}
                    </TableBody>
                </Table>
            </TableContainer>
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
    );
};
