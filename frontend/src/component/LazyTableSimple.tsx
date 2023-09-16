import React, { useEffect, useMemo, useState } from 'react';
import { LazyResult } from '../api';
import {
    compare,
    HeadingCell,
    Order,
    RowsPerPage,
    stableSort
} from './DataTable';
import Stack from '@mui/material/Stack';
import { LazyTable } from './LazyTable';
import { TablePagination } from '@mui/material';
import { LoadingPlaceholder } from './LoadingPlaceholder';
import { logErr } from '../util/errors';

export interface LazyFetchOpts<T> {
    column: keyof T;
    order: Order;
    page: number;
}

interface LazyTableSimpleProps<T> {
    fetchData: (opts: LazyFetchOpts<T>) => Promise<LazyResult<T>>;
    columns: HeadingCell<T>[];
    defaultSortColumn: keyof T;
    defaultSortDir?: Order;
    defaultRowsPerPage?: RowsPerPage;
    paged?: boolean;
    showPager?: boolean;
}

/**
 * Provides a slightly higher level "managed" table that can be used for simple use cases. If advanced filtering
 * is required, you should use the LazyTable directly for more control.
 */
export const LazyTableSimple = <T,>({
    fetchData,
    columns,
    defaultSortColumn,
    defaultSortDir = 'desc',
    paged = false,
    showPager = true,
    defaultRowsPerPage = RowsPerPage.TwentyFive
}: LazyTableSimpleProps<T>) => {
    const [data, setData] = useState<T[]>([]);
    const [count, setCount] = useState(0);
    const [loading, setLoading] = useState(false);
    const [sortOrder, setSortOrder] = useState<Order>(defaultSortDir);
    const [sortColumn, setSortColumn] = useState<keyof T>(defaultSortColumn);
    const [page, setPage] = useState(0);
    const [hasLoaded, setHasLoaded] = useState(false);
    const [rowsPerPage, setRowsPerPage] = useState(defaultRowsPerPage);

    useEffect(() => {
        setLoading(true);
        if (!paged && hasLoaded) {
            setData(data);
            setLoading(false);
            return;
        }
        fetchData({ column: sortColumn, order: sortOrder, page: page })
            .then((resp) => {
                setData(resp.data);
                setCount(resp.count);
            })
            .catch((e) => {
                logErr(e);
            })
            .finally(() => {
                setLoading(false);
                setHasLoaded(true);
            });
    }, [data, fetchData, hasLoaded, page, paged, sortColumn, sortOrder]);

    const rows = useMemo(() => {
        return stableSort(data ?? [], compare(sortOrder, sortColumn)).slice(
            page * rowsPerPage,
            page * rowsPerPage + rowsPerPage
        );
    }, [data, page, rowsPerPage, sortColumn, sortOrder]);

    return loading ? (
        <LoadingPlaceholder />
    ) : (
        <Stack>
            <LazyTable<T>
                columns={columns}
                sortOrder={sortOrder}
                sortColumn={sortColumn}
                onSortColumnChanged={async (column) => {
                    setSortColumn(column);
                }}
                onSortOrderChanged={async (direction) => {
                    setSortOrder(direction);
                }}
                rows={rows}
            />
            {showPager && (
                <Stack direction={'row-reverse'}>
                    <TablePagination
                        page={page}
                        count={count}
                        showFirstButton
                        showLastButton
                        rowsPerPage={rowsPerPage}
                        onRowsPerPageChange={(event) => {
                            setRowsPerPage(parseInt(event.target.value));
                        }}
                        onPageChange={(_, newPage) => {
                            setPage(newPage);
                        }}
                    />
                </Stack>
            )}
        </Stack>
    );
};
