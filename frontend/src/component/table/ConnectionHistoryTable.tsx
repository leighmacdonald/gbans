import { ChangeEvent, useEffect, useState } from 'react';
import {
    apiGetConnections,
    PersonConnection,
    PersonConnectionQuery
} from '../../api';
import { logErr } from '../../util/errors';
import { Order, RowsPerPage } from '../../util/table.ts';
import { LoadingPlaceholder } from '../LoadingPlaceholder';
import { LazyTable } from './LazyTable';
import { connectionColumns } from './connectionColumns.tsx';

export const ConnectionHistoryTable = ({ steam_id }: { steam_id?: string }) => {
    const [bans, setBans] = useState<PersonConnection[]>([]);
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] = useState<keyof PersonConnection>(
        'person_connection_id'
    );
    const [loading, setLoading] = useState(false);
    const [rowPerPageCount, setRowPerPageCount] = useState<number>(
        RowsPerPage.Ten
    );
    const [page, setPage] = useState(0);
    const [totalRows, setTotalRows] = useState<number>(0);

    useEffect(() => {
        const abortController = new AbortController();
        const opts: PersonConnectionQuery = {
            limit: rowPerPageCount,
            offset: page * rowPerPageCount,
            order_by: sortColumn,
            desc: sortOrder == 'desc',
            source_id: steam_id
        };
        setLoading(true);
        apiGetConnections(opts, abortController)
            .then((resp) => {
                setBans(resp.data);
                setTotalRows(resp.count);
                if (page * rowPerPageCount > resp.count) {
                    setPage(0);
                }
            })
            .catch((e) => {
                logErr(e);
            })
            .finally(() => {
                setLoading(false);
            });
        return () => abortController.abort();
    }, [page, rowPerPageCount, sortColumn, sortOrder, steam_id]);

    if (loading) {
        return <LoadingPlaceholder />;
    }
    return (
        <LazyTable<PersonConnection>
            showPager={true}
            count={totalRows}
            rows={bans}
            page={page}
            rowsPerPage={rowPerPageCount}
            sortOrder={sortOrder}
            sortColumn={sortColumn}
            onSortColumnChanged={async (column) => {
                setSortColumn(column);
            }}
            onSortOrderChanged={async (direction) => {
                setSortOrder(direction);
            }}
            onPageChange={(_, newPage: number) => {
                setPage(newPage);
            }}
            onRowsPerPageChange={(
                event: ChangeEvent<HTMLInputElement | HTMLTextAreaElement>
            ) => {
                setRowPerPageCount(parseInt(event.target.value, 10));
                setPage(0);
            }}
            columns={connectionColumns}
        />
    );
};
