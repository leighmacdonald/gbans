import React, { useCallback, useEffect, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import EditIcon from '@mui/icons-material/Edit';
import UndoIcon from '@mui/icons-material/Undo';
import ButtonGroup from '@mui/material/ButtonGroup';
import IconButton from '@mui/material/IconButton';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { formatDuration, intervalToDuration } from 'date-fns';
import format from 'date-fns/format';
import { apiGetBansGroups, BanQueryFilter, IAPIBanGroupRecord } from '../api';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { logErr } from '../util/errors';
import { Order, RowsPerPage } from './DataTable';
import { LazyTable } from './LazyTable';
import { ModalBanGroup, ModalUnbanGroup } from './modal';
import { BanGroupModalProps } from './modal/BanGroupModal';

export const BanGroupTable = () => {
    const [bans, setBans] = useState<IAPIBanGroupRecord[]>([]);
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] =
        useState<keyof IAPIBanGroupRecord>('ban_group_id');
    const [rowPerPageCount, setRowPerPageCount] = useState<number>(
        RowsPerPage.TwentyFive
    );
    const [page, setPage] = useState(0);
    const [totalRows, setTotalRows] = useState<number>(0);
    const { sendFlash } = useUserFlashCtx();

    const onEditGroup = useCallback(
        async (existing: IAPIBanGroupRecord) => {
            try {
                await NiceModal.show<IAPIBanGroupRecord, BanGroupModalProps>(
                    ModalBanGroup,
                    {
                        existing
                    }
                );
                sendFlash('success', 'Updated steam group ban successfully');
            } catch (e) {
                sendFlash('error', `Failed to update steam group ban: ${e}`);
            }
        },
        [sendFlash]
    );

    const onUnbanGroup = useCallback(
        async (ban_group_id: number) => {
            try {
                await NiceModal.show(ModalUnbanGroup, {
                    banId: ban_group_id
                });
                sendFlash('success', 'Unbanned Group successfully');
            } catch (e) {
                sendFlash('error', `Failed to unban Group: ${e}`);
            }
        },
        [sendFlash]
    );

    useEffect(() => {
        const abortController = new AbortController();
        const opts: BanQueryFilter<IAPIBanGroupRecord> = {
            limit: rowPerPageCount,
            offset: page * rowPerPageCount,
            order_by: sortColumn,
            desc: sortOrder == 'desc'
        };
        apiGetBansGroups(opts, abortController)
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
    }, [page, rowPerPageCount, sortColumn, sortOrder]);

    return (
        <LazyTable<IAPIBanGroupRecord>
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
                    label: '#',
                    tooltip: 'Ban ID',
                    sortKey: 'ban_group_id',
                    sortable: true,
                    align: 'left',
                    queryValue: (o) => `${o.ban_group_id}`,
                    renderer: (obj) => (
                        <Typography variant={'body1'}>
                            #{obj.ban_group_id.toString()}
                        </Typography>
                    )
                },
                {
                    label: 'SteamID',
                    tooltip: 'SteamID of the primary target',
                    sortKey: 'target_id',
                    sortable: true,
                    align: 'left',
                    queryValue: (o) => `${o.target_id}`,
                    renderer: (row) => (
                        <Typography variant={'body1'}>
                            {row.target_id}
                        </Typography>
                    )
                },
                {
                    label: 'GroupID',
                    tooltip: 'GroupID',
                    sortKey: 'group_id',
                    sortable: true,
                    align: 'left',
                    queryValue: (o) => `${o.target_id}`,
                    renderer: (row) => (
                        <Typography variant={'body1'}>
                            {row.group_id}
                        </Typography>
                    )
                },
                {
                    label: 'Note',
                    tooltip: 'Mod Note',
                    sortKey: 'note',
                    sortable: false,
                    align: 'left',
                    queryValue: (row) => row.note,
                    renderer: (row) => (
                        <Typography variant={'body1'}>{row.note}</Typography>
                    )
                },
                {
                    label: 'Created',
                    tooltip: 'Created On',
                    sortType: 'date',
                    align: 'left',
                    width: '150px',
                    virtual: true,
                    virtualKey: 'created_on',
                    renderer: (obj) => {
                        return (
                            <Typography variant={'body1'}>
                                {format(obj.created_on, 'yyyy-MM-dd')}
                            </Typography>
                        );
                    }
                },
                {
                    label: 'Expires',
                    tooltip: 'Valid Until',
                    sortType: 'date',
                    align: 'left',
                    width: '150px',
                    virtual: true,
                    virtualKey: 'valid_until',
                    sortable: true,
                    renderer: (obj) => {
                        return (
                            <Typography variant={'body1'}>
                                {format(obj.valid_until, 'yyyy-MM-dd')}
                            </Typography>
                        );
                    }
                },
                {
                    label: 'Duration',
                    tooltip: 'Total Ban Duration',
                    sortType: 'number',
                    align: 'left',
                    width: '150px',
                    virtual: true,
                    virtualKey: 'duration',
                    renderer: (row) => {
                        const dur = intervalToDuration({
                            start: row.created_on,
                            end: row.valid_until
                        });
                        const durationText =
                            dur.years && dur.years > 5
                                ? 'Permanent'
                                : formatDuration(dur);
                        return (
                            <Typography variant={'body1'} overflow={'hidden'}>
                                {durationText}
                            </Typography>
                        );
                    }
                },
                {
                    label: 'Act.',
                    tooltip: 'Actions',
                    sortKey: 'reason',
                    sortable: false,
                    align: 'left',
                    renderer: (row) => (
                        <ButtonGroup fullWidth>
                            <IconButton
                                color={'warning'}
                                onClick={async () => {
                                    await onEditGroup(row);
                                }}
                            >
                                <Tooltip title={'Edit Ban'}>
                                    <EditIcon />
                                </Tooltip>
                            </IconButton>
                            <IconButton
                                color={'success'}
                                onClick={async () => {
                                    await onUnbanGroup(row.ban_group_id);
                                }}
                            >
                                <Tooltip title={'Remove Ban'}>
                                    <UndoIcon />
                                </Tooltip>
                            </IconButton>
                        </ButtonGroup>
                    )
                }
            ]}
        />
    );
};
