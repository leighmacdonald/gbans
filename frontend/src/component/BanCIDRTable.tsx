import React, { useCallback, useEffect, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import EditIcon from '@mui/icons-material/Edit';
import UndoIcon from '@mui/icons-material/Undo';
import ButtonGroup from '@mui/material/ButtonGroup';
import IconButton from '@mui/material/IconButton';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { formatDuration, intervalToDuration } from 'date-fns';
import { Formik } from 'formik';
import IPCIDR from 'ip-cidr';
import * as yup from 'yup';
import {
    apiGetBansCIDR,
    BanReason,
    CIDRBanRecord,
    BanCIDRQueryFilter
} from '../api';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { logErr } from '../util/errors';
import { DataTableRelativeDateField } from './DataTableRelativeDateField';
import { LazyTable, Order, RowsPerPage } from './LazyTable';
import { PersonCell } from './PersonCell';
import { TableCellBool } from './TableCellBool';
import { DeletedField, deletedValidator } from './formik/DeletedField';
import { FilterButtons } from './formik/FilterButtons';
import { IPField, ipFieldValidator } from './formik/IPField';
import { SourceIdField, sourceIdValidator } from './formik/SourceIdField';
import { TargetIDField, targetIdValidator } from './formik/TargetIdField';
import { ModalBanCIDR, ModalUnbanCIDR } from './modal';
import { BanCIDRModalProps } from './modal/BanCIDRModal';

interface CIDRBanFilterValues {
    ip: string;
    source_id: string;
    target_id: string;
    deleted: boolean;
}

const validationSchema = yup.object({
    ip: ipFieldValidator,
    source_id: sourceIdValidator,
    target_id: targetIdValidator,
    deleted: deletedValidator
});

export const BanCIDRTable = () => {
    const [bans, setBans] = useState<CIDRBanRecord[]>([]);
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] = useState<keyof CIDRBanRecord>('net_id');
    const [rowPerPageCount, setRowPerPageCount] = useState<number>(
        RowsPerPage.TwentyFive
    );
    const [page, setPage] = useState(0);
    const [totalRows, setTotalRows] = useState<number>(0);
    const [source, setSource] = useState('');
    const [target, setTarget] = useState('');
    const [deleted, setDeleted] = useState(false);
    const [ip, setIp] = useState('');
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
        async (existing: CIDRBanRecord) => {
            try {
                await NiceModal.show<CIDRBanRecord, BanCIDRModalProps>(
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
        const opts: BanCIDRQueryFilter = {
            limit: rowPerPageCount,
            offset: page * rowPerPageCount,
            order_by: sortColumn,
            desc: sortOrder == 'desc',
            source_id: source,
            target_id: target,
            deleted: deleted,
            ip: ip
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
    }, [
        source,
        deleted,
        page,
        rowPerPageCount,
        sortColumn,
        sortOrder,
        target,
        ip
    ]);

    const iv: CIDRBanFilterValues = {
        ip: '',
        source_id: '',
        target_id: '',
        deleted: false
    };

    const onSubmit = useCallback((values: CIDRBanFilterValues) => {
        setSource(values.source_id);
        setTarget(values.target_id);
        setDeleted(values.deleted);
        setIp(values.ip);
    }, []);

    const onReset = useCallback(() => {
        setSource(iv.source_id);
        setTarget(iv.target_id);
        setDeleted(iv.deleted);
        setIp(iv.ip);
    }, [iv.source_id, iv.target_id, iv.deleted, iv.ip]);

    return (
        <Formik
            onSubmit={onSubmit}
            initialValues={iv}
            onReset={onReset}
            validationSchema={validationSchema}
            validateOnChange={true}
            validateOnBlur={true}
        >
            <Grid container spacing={3}>
                <Grid xs={12}>
                    <Grid container spacing={2}>
                        <Grid xs>
                            <SourceIdField />
                        </Grid>
                        <Grid xs>
                            <TargetIDField />
                        </Grid>
                        <Grid xs>
                            <IPField />
                        </Grid>
                        <Grid xs>
                            <DeletedField />
                        </Grid>
                        <Grid xs>
                            <FilterButtons />
                        </Grid>
                    </Grid>
                </Grid>
                <Grid xs={12}>
                    <LazyTable<CIDRBanRecord>
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
                            event: React.ChangeEvent<
                                HTMLInputElement | HTMLTextAreaElement
                            >
                        ) => {
                            setRowPerPageCount(
                                parseInt(event.target.value, 10)
                            );
                            setPage(0);
                        }}
                        columns={[
                            {
                                label: '#',
                                tooltip: 'Ban ID',
                                sortKey: 'net_id',
                                sortable: true,
                                align: 'left',
                                renderer: (obj) => (
                                    <Typography variant={'body1'}>
                                        #{obj.net_id.toString()}
                                    </Typography>
                                )
                            },
                            {
                                label: 'A',
                                tooltip: ' BanAuthor',
                                sortKey: 'source_id',
                                sortable: true,
                                align: 'left',
                                renderer: (row) => (
                                    <PersonCell
                                        steam_id={row.source_id}
                                        personaname={''}
                                        avatar_hash={row.source_avatarhash}
                                    />
                                )
                            },
                            {
                                label: 'Target',
                                tooltip: 'Steam Name',
                                sortKey: 'target_id',
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
                                label: 'CIDR',
                                tooltip: 'CIDR Range',
                                sortKey: 'cidr',
                                sortable: true,
                                align: 'left',
                                renderer: (obj) => {
                                    try {
                                        return (
                                            <Typography variant={'body1'}>
                                                {obj.cidr}
                                            </Typography>
                                        );
                                    } catch (e) {
                                        return <>?</>;
                                    }
                                }
                            },
                            {
                                label: 'Hosts',
                                tooltip: 'Total hosts in CIDR range',
                                sortable: false,
                                align: 'left',
                                renderer: (obj) => {
                                    try {
                                        const network = new IPCIDR(obj.cidr);
                                        const hosts = network.toArray().length;
                                        return (
                                            <Typography variant={'body1'}>
                                                {hosts}
                                            </Typography>
                                        );
                                    } catch (e) {
                                        logErr(e);
                                    }
                                    return (
                                        <Typography variant={'body1'}>
                                            ?
                                        </Typography>
                                    );
                                }
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
                                align: 'left',
                                renderer: (row) => (
                                    <Typography variant={'body1'}>
                                        {row.reason_text}
                                    </Typography>
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
                                        <Typography
                                            variant={'body1'}
                                            overflow={'hidden'}
                                        >
                                            {durationText}
                                        </Typography>
                                    );
                                }
                            },
                            {
                                label: 'A',
                                tooltip:
                                    'Is this ban active (not deleted/inactive/unbanned)',
                                align: 'center',
                                width: '50px',
                                sortKey: 'deleted',
                                renderer: (row) => (
                                    <TableCellBool enabled={!row.deleted} />
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
                </Grid>
            </Grid>
        </Formik>
    );
};
