import React, { useCallback, useEffect, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import EditIcon from '@mui/icons-material/Edit';
import UndoIcon from '@mui/icons-material/Undo';
import ButtonGroup from '@mui/material/ButtonGroup';
import IconButton from '@mui/material/IconButton';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { formatDuration, intervalToDuration } from 'date-fns';
import {
    apiGetBansCIDR,
    BanQueryFilter,
    BanReason,
    IAPIBanCIDRRecord
} from '../api';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { logErr } from '../util/errors';
import { steamIdQueryValue } from '../util/text';
import { Order, RowsPerPage } from './DataTable';
import { DataTableRelativeDateField } from './DataTableRelativeDateField';
import { LazyTable } from './LazyTable';
import { ModalBanCIDR, ModalUnbanCIDR } from './modal';
import { BanCIDRModalProps } from './modal/BanCIDRModal';

export const BanCIDRTable = () => {
    const [bans, setBans] = useState<IAPIBanCIDRRecord[]>([]);
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] =
        useState<keyof IAPIBanCIDRRecord>('net_id');
    const [rowPerPageCount, setRowPerPageCount] = useState<number>(
        RowsPerPage.TwentyFive
    );
    const [page, setPage] = useState(0);
    const [totalRows, setTotalRows] = useState<number>(0);
    const { sendFlash } = useUserFlashCtx();

    const onUnbanCIDR = useCallback(
        async (net_id: number) => {
            try {
                await NiceModal.show(ModalUnbanCIDR, {
                    banId: net_id
                });
                sendFlash('success', 'Unbanned CIDR successfully');
            } catch (e) {
                sendFlash('error', `Failed to unban: ${e}`);
            }
        },
        [sendFlash]
    );

    const onEditCIDR = useCallback(
        async (existing: IAPIBanCIDRRecord) => {
            try {
                await NiceModal.show<IAPIBanCIDRRecord, BanCIDRModalProps>(
                    ModalBanCIDR,
                    {
                        existing
                    }
                );
                sendFlash('success', 'Updated CIDR ban successfully');
            } catch (e) {
                sendFlash('error', `Failed to update ban: ${e}`);
            }
        },
        [sendFlash]
    );

    useEffect(() => {
        const abortController = new AbortController();
        const opts: BanQueryFilter<IAPIBanCIDRRecord> = {
            limit: rowPerPageCount,
            offset: page * rowPerPageCount,
            order_by: sortColumn,
            desc: sortOrder == 'desc'
        };
        apiGetBansCIDR(opts, abortController)
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
        <LazyTable<IAPIBanCIDRRecord>
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
                    sortKey: 'net_id',
                    sortable: true,
                    align: 'left',
                    queryValue: (o) => `${o.net_id}`,
                    renderer: (obj) => (
                        <Typography variant={'body1'}>
                            #{obj.net_id.toString()}
                        </Typography>
                    )
                },
                {
                    label: 'Author',
                    tooltip: 'Author ID',
                    sortKey: 'source_id',
                    sortable: true,
                    align: 'left',
                    queryValue: (o) => steamIdQueryValue(o.source_id),
                    renderer: (obj) => (
                        <Typography variant={'body1'}>
                            {obj.source_id.toString()}
                        </Typography>
                    )
                },
                {
                    label: 'Target',
                    tooltip: 'Target SID',
                    sortKey: 'target_id',
                    sortable: true,
                    align: 'left',
                    queryValue: (o) => steamIdQueryValue(o.target_id),
                    renderer: (obj) => (
                        <Typography variant={'body1'}>
                            {obj.target_id.toString()}
                        </Typography>
                    )
                },
                {
                    label: 'CIDR',
                    tooltip: 'CIDR Range',
                    sortKey: 'cidr',
                    sortable: true,
                    align: 'left',
                    queryValue: (o) => `${o.target_id}`,
                    renderer: (obj) => {
                        try {
                            return (
                                <Typography variant={'body1'}>
                                    {obj.cidr.IP}
                                </Typography>
                            );
                        } catch (e) {
                            return <>?</>;
                        }
                    }
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
                            <DataTableRelativeDateField
                                date={obj.created_on}
                                suffix={true}
                            />
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
                            <DataTableRelativeDateField
                                date={obj.valid_until}
                            />
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
                                    await onEditCIDR(row);
                                }}
                            >
                                <Tooltip title={'Edit CIDR Ban'}>
                                    <EditIcon />
                                </Tooltip>
                            </IconButton>
                            <IconButton
                                color={'success'}
                                onClick={async () => {
                                    await onUnbanCIDR(row.net_id);
                                }}
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
