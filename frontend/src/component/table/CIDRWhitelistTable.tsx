import React, { useMemo, useState } from 'react';
import { CIDRBlockWhitelist, MatchSummary } from '../../api';
import { LazyTable, Order, RowsPerPage } from './LazyTable';
import { compare, stableSort } from './LazyTableSimple';

export const CIDRWhitelistTable = ({
    rows
}: {
    rows: CIDRBlockWhitelist[];
}) => {
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] =
        useState<keyof CIDRBlockWhitelist>('address');
    const [totalRows, setTotalRows] = useState<number>(0);
    const [page, setPage] = useState(0);
    const [rowPerPageCount, setRowPerPageCount] = useState<number>(
        RowsPerPage.TwentyFive
    );

    const sorted = useMemo(() => {
        return stableSort(rows, compare(sortOrder, sortColumn)).slice(
            (page - 1) * rowPerPageCount,
            (page - 1) * rowPerPageCount + rowPerPageCount
        );
    }, [rows, page, sortColumn, sortOrder]);

    return (
        <LazyTable<CIDRBlockWhitelist>
            columns={[
                {
                    label: 'Address',
                    sortable: true,
                    sortKey: 'address',
                    tooltip: 'Whitelisted IP Address'
                }
            ]}
            sortOrder={sortOrder}
            sortColumn={sortColumn}
            onSortColumnChanged={async (column) => {
                setSortColumn(column);
            }}
            onSortOrderChanged={async (direction) => {
                setSortOrder(direction);
            }}
            rows={sorted}
        />
    );
};
