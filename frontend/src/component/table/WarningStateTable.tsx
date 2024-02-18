import { ChangeEvent, useMemo, useState } from 'react';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { Filter, filterActionString, UserWarning } from '../../api/filters';
import { compare, Order, RowsPerPage, stableSort } from '../../util/table.ts';
import { renderDateTime } from '../../util/text.tsx';
import { PersonCell } from '../PersonCell';
import { LazyTable } from './LazyTable';

export const WarningStateTable = ({
    warnings
}: {
    warnings: UserWarning[];
}) => {
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] =
        useState<keyof UserWarning>('created_on');
    const [page, setPage] = useState(0);
    const [rowPerPageCount, setRowPerPageCount] = useState<number>(
        RowsPerPage.Fifty
    );

    const sorted = useMemo(() => {
        return stableSort(warnings, compare(sortOrder, sortColumn)).slice(
            page * rowPerPageCount,
            page * rowPerPageCount + rowPerPageCount
        );
    }, [warnings, sortOrder, sortColumn, page, rowPerPageCount]);

    const renderFilter = (f: Filter) => {
        const pat = f.is_regex ? (f.pattern as string) : (f.pattern as string);

        return (
            <>
                <Typography variant={'h6'}>
                    Matched {f.is_regex ? 'Regex' : 'Text'}
                </Typography>
                <Typography variant={'body1'}>{pat}</Typography>
                <Typography variant={'body1'}>Weight: {f.weight}</Typography>
                <Typography variant={'body1'}>
                    Action: {filterActionString(f.action)}
                </Typography>
            </>
        );
    };

    return (
        <LazyTable
            loading={false}
            rows={sorted}
            rowsPerPage={rowPerPageCount}
            page={page}
            showPager={true}
            count={sorted.length}
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
                    tooltip: 'Player',
                    label: 'Player',
                    sortKey: 'personaname',
                    align: 'left',
                    width: '150px',
                    renderer: (row) => {
                        return (
                            <PersonCell
                                personaname={row.personaname}
                                steam_id={row.steam_id}
                                avatar_hash={row.avatar}
                            />
                        );
                    }
                },
                {
                    tooltip: 'Created On',
                    label: 'Created On',
                    sortKey: 'created_on',
                    align: 'left',
                    width: '150px',
                    hideSm: true,
                    renderer: (row) => {
                        return renderDateTime(row.created_on);
                    }
                },

                {
                    tooltip: 'Server',
                    label: 'Server',
                    sortKey: 'server_name',
                    align: 'left',
                    width: '100px',
                    renderer: (row) => {
                        return (
                            <Typography variant={'body2'}>
                                {row.server_name}
                            </Typography>
                        );
                    }
                },
                {
                    tooltip: 'Matched',
                    label: 'Matched',
                    align: 'left',
                    sortKey: 'matched',
                    width: '50px',
                    renderer: (row) => {
                        return (
                            <Tooltip title={renderFilter(row.matched_filter)}>
                                <Typography
                                    variant={'body1'}
                                    sx={{
                                        textDecoration: 'underline',
                                        cursor: 'help'
                                    }}
                                >
                                    {row.matched}
                                </Typography>
                            </Tooltip>
                        );
                    }
                },
                {
                    tooltip: 'Total sum of all matched weights',
                    label: 'Sum',
                    align: 'left',
                    sortKey: 'current_total',
                    width: '50px',
                    renderer: (row) => {
                        return (
                            <Typography variant={'body1'}>
                                {row.current_total}
                            </Typography>
                        );
                    }
                },
                {
                    tooltip: 'Message',
                    label: 'Message',
                    sortKey: 'message',
                    align: 'left',
                    renderer: (obj) => {
                        return obj.message;
                    }
                }
            ]}
        />
    );
};
