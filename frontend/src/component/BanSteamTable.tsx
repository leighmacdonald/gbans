import React, { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import NiceModal from '@ebay/nice-modal-react';
import EditIcon from '@mui/icons-material/Edit';
import UndoIcon from '@mui/icons-material/Undo';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import IconButton from '@mui/material/IconButton';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import format from 'date-fns/format';
import {
    apiGetBansSteam,
    BanQueryFilter,
    BanReason,
    SteamBanRecord
} from '../api';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { logErr } from '../util/errors';
import { Order, RowsPerPage } from './DataTable';
import {
    DataTableRelativeDateField,
    isPermanentBan
} from './DataTableRelativeDateField';
import { LazyTable } from './LazyTable';
import { PersonCell } from './PersonCell';
import { TableCellLink } from './TableCellLink';
import { ModalBanSteam, ModalUnbanSteam } from './modal';

export const BanSteamTable = () => {
    const [bans, setBans] = useState<SteamBanRecord[]>([]);
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] =
        useState<keyof SteamBanRecord>('ban_id');
    const [rowPerPageCount, setRowPerPageCount] = useState<number>(
        RowsPerPage.TwentyFive
    );
    const [page, setPage] = useState(0);
    const [totalRows, setTotalRows] = useState<number>(0);
    const { sendFlash } = useUserFlashCtx();
    const navigate = useNavigate();

    const onUnbanSteam = useCallback(
        async (ban: SteamBanRecord) => {
            try {
                await NiceModal.show(ModalUnbanSteam, {
                    banId: ban.ban_id,
                    personaName: ban.target_personaname
                });
                sendFlash('success', 'Unbanned successfully');
            } catch (e) {
                sendFlash('error', `Failed to unban: ${e}`);
            }
        },
        [sendFlash]
    );

    const onEditSteam = useCallback(
        async (ban: SteamBanRecord) => {
            try {
                await NiceModal.show(ModalBanSteam, {
                    banId: ban.ban_id,
                    personaName: ban.target_personaname,
                    existing: ban
                });
                sendFlash('success', 'Updated ban successfully');
            } catch (e) {
                sendFlash('error', `Failed to update ban: ${e}`);
            }
        },
        [sendFlash]
    );

    useEffect(() => {
        const abortController = new AbortController();
        const opts: BanQueryFilter<SteamBanRecord> = {
            limit: rowPerPageCount,
            offset: page * rowPerPageCount,
            order_by: sortColumn,
            desc: sortOrder == 'desc'
        };
        apiGetBansSteam(opts, abortController)
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
                event: React.ChangeEvent<HTMLInputElement | HTMLTextAreaElement>
            ) => {
                setRowPerPageCount(parseInt(event.target.value, 10));
                setPage(0);
            }}
            columns={[
                {
                    label: '#',
                    tooltip: 'Ban ID',
                    sortKey: 'ban_id',
                    sortable: true,
                    align: 'left',
                    renderer: (row) => (
                        <TableCellLink
                            label={`#${row.ban_id.toString()}`}
                            to={`/ban/${row.ban_id}`}
                        />
                    )
                },
                {
                    label: 'A',
                    tooltip: 'Ban Author',
                    sortKey: 'source_personaname',
                    sortable: true,
                    align: 'center',
                    renderer: (row) => (
                        <Tooltip title={row.source_personaname}>
                            <PersonCell
                                steam_id={row.source_id}
                                personaname={''}
                                avatar_hash={row.source_avatarhash}
                            />
                        </Tooltip>
                    )
                },
                {
                    label: 'Target',
                    tooltip: 'Steam Name',
                    sortKey: 'target_personaname',
                    sortable: true,
                    align: 'left',
                    renderer: (row) => (
                        <PersonCell
                            steam_id={row.target_id}
                            personaname={row.target_personaname}
                            avatar_hash={row.target_avatarhash}
                        />
                    )
                },
                {
                    label: 'Reason',
                    tooltip: 'Reason',
                    sortKey: 'reason',
                    sortable: true,
                    align: 'left',
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
                    align: 'left'
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
                        return isPermanentBan(
                            row.created_on,
                            row.valid_until
                        ) ? (
                            'Permanent'
                        ) : (
                            <DataTableRelativeDateField
                                date={row.created_on}
                                compareDate={row.valid_until}
                            />
                        );
                    }
                },
                {
                    label: 'FL',
                    tooltip: 'Are friends also included in the ban',
                    align: 'left',
                    width: '150px',
                    sortKey: 'include_friends',
                    renderer: (row) => {
                        return (
                            <Typography variant={'body1'}>
                                {row.include_friends ? 'yes' : 'no'}
                            </Typography>
                        );
                    }
                },
                {
                    label: 'Rep.',
                    tooltip: 'Report',
                    sortable: false,
                    align: 'left',
                    width: '20px',
                    queryValue: (o) => `${o.report_id}`,
                    renderer: (row) =>
                        row.report_id > 0 ? (
                            <Tooltip title={'View Report'}>
                                <Button
                                    variant={'text'}
                                    onClick={() => {
                                        navigate(`/report/${row.report_id}`);
                                    }}
                                >
                                    #{row.report_id}
                                </Button>
                            </Tooltip>
                        ) : (
                            <></>
                        )
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
                                    await onEditSteam(row);
                                }}
                            >
                                <Tooltip title={'Edit Ban'}>
                                    <EditIcon />
                                </Tooltip>
                            </IconButton>
                            <IconButton
                                color={'success'}
                                onClick={async () => {
                                    await onUnbanSteam(row);
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
