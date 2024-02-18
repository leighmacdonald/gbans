import {
    ChangeEvent,
    Dispatch,
    SetStateAction,
    useEffect,
    useState
} from 'react';
import Typography from '@mui/material/Typography';
import { apiGetBansSteam, BanReasons, SteamBanRecord } from '../../api';
import { logErr } from '../../util/errors';
import { Order, RowsPerPage } from '../../util/table.ts';
import { PersonCell } from '../PersonCell';
import { LazyTable } from './LazyTable';
import { TableCellBool } from './TableCellBool';

export const BanHistoryTable = ({
    steam_id,
    setBanCount
}: {
    steam_id?: string;
    setBanCount: Dispatch<SetStateAction<number>>;
}) => {
    const [bans, setBans] = useState<SteamBanRecord[]>([]);
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] =
        useState<keyof SteamBanRecord>('ban_id');
    const [rowPerPageCount, setRowPerPageCount] = useState<number>(
        RowsPerPage.Ten
    );
    const [page, setPage] = useState(0);
    const [totalRows, setTotalRows] = useState<number>(0);

    useEffect(() => {
        const abortController = new AbortController();
        apiGetBansSteam(
            {
                limit: rowPerPageCount,
                offset: page * rowPerPageCount,
                order_by: sortColumn,
                desc: sortOrder == 'desc',
                target_id: steam_id,
                deleted: true
            },
            abortController
        )
            .then((resp) => {
                setBans(resp.data);
                setTotalRows(resp.count);
                setBanCount(resp.count);
                if (page * rowPerPageCount > resp.count) {
                    setPage(0);
                }
            })
            .catch((e) => {
                logErr(e);
            });
        return () => abortController.abort();
    }, [page, rowPerPageCount, setBanCount, sortColumn, sortOrder, steam_id]);

    return (
        <LazyTable<SteamBanRecord>
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
            columns={[
                {
                    label: 'A',
                    tooltip:
                        'Is this ban active (not deleted/inactive/unbanned)',
                    align: 'center',
                    width: '50px',
                    sortKey: 'deleted',
                    renderer: (row) => <TableCellBool enabled={!row.deleted} />
                },
                {
                    label: 'Created',
                    tooltip: 'Created On',
                    sortKey: 'created_on',
                    sortType: 'date',
                    sortable: true,
                    align: 'left',
                    width: '150px'
                },
                {
                    label: 'Expires',
                    tooltip: 'Expires',
                    sortKey: 'valid_until',
                    sortType: 'date',
                    sortable: true,
                    align: 'left'
                },
                {
                    label: 'Ban Author',
                    tooltip: 'Ban Author',
                    sortKey: 'source_id',
                    sortType: 'string',
                    align: 'left',
                    width: '150px',
                    renderer: (row) => (
                        <PersonCell
                            steam_id={row.source_id}
                            personaname={row.source_personaname}
                            avatar_hash={row.source_avatarhash}
                        />
                    )
                },
                {
                    label: 'Reason',
                    tooltip: 'Reason',
                    sortKey: 'reason',
                    sortable: true,
                    sortType: 'string',
                    align: 'left',
                    renderer: (row) => (
                        <Typography variant={'body1'}>
                            {BanReasons[row.reason]}
                        </Typography>
                    )
                },
                {
                    label: 'Custom',
                    tooltip: 'Custom Reason',
                    sortKey: 'reason_text',
                    sortType: 'string',
                    align: 'left'
                },
                {
                    label: 'Unban Reason',
                    tooltip: 'Unban Reason',
                    sortKey: 'unban_reason_text',
                    sortType: 'string',
                    align: 'left'
                }
            ]}
        />
    );
};
