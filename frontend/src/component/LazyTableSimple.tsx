import React, { useCallback, useEffect, useMemo, useState } from 'react';
import Stack from '@mui/material/Stack';
import { noop } from 'lodash-es';
import { LazyResult } from '../api';
import { logErr } from '../util/errors';
import {
    compare,
    HeadingCell,
    Order,
    RowsPerPage,
    stableSort
} from './DataTable';
import { LazyTable } from './LazyTable';
import { LoadingPlaceholder } from './LoadingPlaceholder';

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
        const abortController = new AbortController();
        const fetchNewData = async () => {
            setLoading(true);
            if (!paged && hasLoaded) {
                setData(data);
                setLoading(false);
                return;
            }
            try {
                const results = await fetchData({
                    column: sortColumn,
                    order: sortOrder,
                    page: page
                });
                setData(results.data);
                setCount(results.count);
            } catch (e) {
                logErr(e);
            } finally {
                setLoading(false);
                setHasLoaded(true);
            }
        };

        fetchNewData().then(noop);

        return () => abortController.abort();
    }, [data, fetchData, hasLoaded, page, paged, sortColumn, sortOrder]);

    const rows = useMemo(() => {
        return stableSort(data ?? [], compare(sortOrder, sortColumn)).slice(
            page * rowsPerPage,
            page * rowsPerPage + rowsPerPage
        );
    }, [data, page, rowsPerPage, sortColumn, sortOrder]);

    const onPagerRowsPerPageChange = useCallback(
        (event: { target: { value: string } }) => {
            setRowsPerPage(parseInt(event.target.value));
        },
        []
    );

    const onPagerRowsChange = useCallback(
        () => (_: never, newPage: number) => {
            setPage(newPage);
        },
        []
    );
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
                page={page}
                count={count}
                rowsPerPage={rowsPerPage}
                rows={rows}
                showPager={showPager}
                onRowsPerPageChange={onPagerRowsPerPageChange}
                onPageChange={onPagerRowsChange}
            />
        </Stack>
    );
};
