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
import {
    apiGetBansASN,
    BanQueryFilter,
    BanReason,
    IAPIBanASNRecord
} from '../api';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { logErr } from '../util/errors';
import { Order, RowsPerPage } from './DataTable';
import { LazyTable } from './LazyTable';
import { ModalBanASN, ModalUnbanASN } from './modal';
import { BanASNModalProps } from './modal/BanASNModal';

export const BanASNTable = () => {
    const [bans, setBans] = useState<IAPIBanASNRecord[]>([]);
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] =
        useState<keyof IAPIBanASNRecord>('ban_asn_id');
    const [rowPerPageCount, setRowPerPageCount] = useState<number>(
        RowsPerPage.TwentyFive
    );
    const [page, setPage] = useState(0);
    const [totalRows, setTotalRows] = useState<number>(0);
    const { sendFlash } = useUserFlashCtx();

    const onUnbanASN = useCallback(
        async (as_num: number) => {
            try {
                await NiceModal.show(ModalUnbanASN, {
                    banId: as_num
                });
                sendFlash('success', 'Unbanned ASN successfully');
            } catch (e) {
                sendFlash('error', `Failed to unban ASN: ${e}`);
            }
        },
        [sendFlash]
    );

    const onEditASN = useCallback(
        async (existing: IAPIBanASNRecord) => {
            try {
                await NiceModal.show<IAPIBanASNRecord, BanASNModalProps>(
                    ModalBanASN,
                    {
                        existing
                    }
                );
                sendFlash('success', 'Updated ASN ban successfully');
            } catch (e) {
                sendFlash('error', `Failed to update ASN ban: ${e}`);
            }
        },
        [sendFlash]
    );

    useEffect(() => {
        const abortController = new AbortController();
        const opts: BanQueryFilter<IAPIBanASNRecord> = {
            limit: rowPerPageCount,
            offset: page * rowPerPageCount,
            order_by: sortColumn,
            desc: sortOrder == 'desc'
        };
        apiGetBansASN(opts, abortController)
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
        <LazyTable<IAPIBanASNRecord>
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
                    sortKey: 'ban_asn_id',
                    sortable: true,
                    align: 'left',
                    queryValue: (o) => `${o.ban_asn_id}`,
                    renderer: (obj) => (
                        <Typography variant={'body1'}>
                            #{obj.ban_asn_id.toString()}
                        </Typography>
                    )
                },
                {
                    label: 'ASN',
                    tooltip: 'Autonomous System Numbers',
                    sortKey: 'as_num',
                    sortable: true,
                    align: 'left',
                    queryValue: (o) => `${o.as_num}`,
                    renderer: (row) => (
                        <Typography variant={'body1'}>{row.as_num}</Typography>
                    )
                },
                {
                    label: 'Reason',
                    tooltip: 'Reason',
                    sortKey: 'reason',
                    sortable: true,
                    align: 'left',
                    queryValue: (o) => BanReason[o.reason],
                    renderer: (row) => (
                        <Typography variant={'body1'}>
                            {BanReason[row.reason]}
                        </Typography>
                    )
                },
                {
                    label: 'Custom Reason',
                    tooltip: 'Custom',
                    sortKey: 'reason_text',
                    sortable: false,
                    align: 'left',
                    queryValue: (o) => o.reason_text
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
                                onClick={async () => await onEditASN(row)}
                            >
                                <Tooltip title={'Edit ASN Ban'}>
                                    <EditIcon />
                                </Tooltip>
                            </IconButton>
                            <IconButton
                                color={'success'}
                                onClick={async () =>
                                    await onUnbanASN(row.as_num)
                                }
                            >
                                <Tooltip title={'Remove CIDR Ban'}>
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
