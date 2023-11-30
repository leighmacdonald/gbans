import React, { useEffect, useState } from 'react';
import Typography from '@mui/material/Typography';
import {
    apiGetConnections,
    PersonConnection,
    PersonConnectionQuery
} from '../../api';
import { logErr } from '../../util/errors';
import { renderDateTime } from '../../util/text';
import { LazyTable, Order, RowsPerPage } from './LazyTable';

export const ConnectionHistoryTable = ({ steam_id }: { steam_id?: string }) => {
    const [bans, setBans] = useState<PersonConnection[]>([]);
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] = useState<keyof PersonConnection>(
        'person_connection_id'
    );
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
            });
        return () => abortController.abort();
    }, [page, rowPerPageCount, sortColumn, sortOrder, steam_id]);

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
                event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>
            ) => {
                setRowPerPageCount(parseInt(event.target.value, 10));
                setPage(0);
            }}
            columns={[
                {
                    label: 'Created',
                    tooltip: 'Created On',
                    sortKey: 'created_on',
                    sortType: 'date',
                    align: 'left',
                    width: '150px',
                    sortable: true,
                    renderer: (obj) => (
                        <Typography variant={'body1'}>
                            {renderDateTime(obj.created_on)}
                        </Typography>
                    )
                },
                {
                    label: 'Name',
                    tooltip: 'Name',
                    sortKey: 'persona_name',
                    sortType: 'string',
                    align: 'left',
                    width: '150px',
                    sortable: true
                },
                {
                    label: 'IP Address',
                    tooltip: 'IP Address',
                    sortKey: 'ip_addr',
                    sortType: 'string',
                    align: 'left',
                    sortable: true
                }
            ]}
        />
    );
};
