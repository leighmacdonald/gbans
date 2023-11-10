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
import format from 'date-fns/format';
import { Formik } from 'formik';
import * as yup from 'yup';
import {
    apiGetBansASN,
    BanReason,
    ASNBanRecord,
    BanASNQueryFilter
} from '../api';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { logErr } from '../util/errors';
import { Order, RowsPerPage } from './DataTable';
import { LazyTable } from './LazyTable';
import { PersonCell } from './PersonCell';
import { TableCellBool } from './TableCellBool';
import { ASNumberField, asNumberFieldValidator } from './formik/ASNumberField';
import { DeletedField, deletedValidator } from './formik/DeletedField';
import { FilterButtons } from './formik/FilterButtons';
import { SourceIdField, sourceIdValidator } from './formik/SourceIdField';
import { TargetIDField, targetIdValidator } from './formik/TargetIdField';
import { ModalBanASN, ModalUnbanASN } from './modal';
import { BanASNModalProps } from './modal/BanASNModal';

interface ASNFilterValues {
    as_num?: number;
    source_id: string;
    target_id: string;
    deleted: boolean;
}

const validationSchema = yup.object({
    as_num: asNumberFieldValidator,
    source_id: sourceIdValidator,
    target_id: targetIdValidator,
    deleted: deletedValidator
});

export const BanASNTable = () => {
    const [bans, setBans] = useState<ASNBanRecord[]>([]);
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] =
        useState<keyof ASNBanRecord>('ban_asn_id');
    const [rowPerPageCount, setRowPerPageCount] = useState<number>(
        RowsPerPage.TwentyFive
    );
    const [page, setPage] = useState(0);
    const [totalRows, setTotalRows] = useState<number>(0);
    const [asNum, setASNum] = useState<number>();
    const [source, setSource] = useState('');
    const [target, setTarget] = useState('');
    const [deleted, setDeleted] = useState(false);
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
        async (existing: ASNBanRecord) => {
            try {
                await NiceModal.show<ASNBanRecord, BanASNModalProps>(
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
        const opts: BanASNQueryFilter = {
            limit: rowPerPageCount,
            offset: page * rowPerPageCount,
            order_by: sortColumn,
            desc: sortOrder == 'desc',
            deleted: deleted,
            source_id: source,
            target_id: target,
            as_num: asNum
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
    }, [
        asNum,
        deleted,
        page,
        rowPerPageCount,
        sortColumn,
        sortOrder,
        source,
        target
    ]);

    const iv: ASNFilterValues = {
        as_num: undefined,
        source_id: '',
        target_id: '',
        deleted: false
    };

    const onSubmit = useCallback((values: ASNFilterValues) => {
        setASNum(values.as_num);
        setSource(values.source_id);
        setTarget(values.target_id);
        setDeleted(values.deleted);
    }, []);

    const onReset = useCallback(() => {
        setASNum(iv.as_num);
        setSource(iv.source_id);
        setTarget(iv.target_id);
        setDeleted(iv.deleted);
    }, [iv.as_num, iv.source_id, iv.deleted, iv.target_id]);

    return (
        <Formik
            initialValues={iv}
            onReset={onReset}
            onSubmit={onSubmit}
            validationSchema={validationSchema}
            validateOnChange={true}
        >
            <Grid container spacing={3}>
                <Grid xs={12}>
                    <Grid container spacing={2}>
                        <Grid xs>
                            <ASNumberField />
                        </Grid>
                        <Grid xs>
                            <SourceIdField />
                        </Grid>
                        <Grid xs>
                            <TargetIDField />
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
                    <LazyTable<ASNBanRecord>
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
                                sortKey: 'ban_asn_id',
                                sortable: true,
                                align: 'left',
                                renderer: (obj) => (
                                    <Typography variant={'body1'}>
                                        #{obj.ban_asn_id.toString()}
                                    </Typography>
                                )
                            },
                            {
                                label: 'A',
                                tooltip: 'Ban Author Name',
                                sortKey: 'source_personaname',
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
                                label: 'Name',
                                tooltip: 'Persona Name',
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
                                label: 'ASN',
                                tooltip: 'Autonomous System Numbers',
                                sortKey: 'as_num',
                                sortable: true,
                                align: 'left',
                                renderer: (row) => (
                                    <Typography variant={'body1'}>
                                        {row.as_num}
                                    </Typography>
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
                                            {format(
                                                obj.created_on,
                                                'yyyy-MM-dd'
                                            )}
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
                                            {format(
                                                obj.valid_until,
                                                'yyyy-MM-dd'
                                            )}
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
                                            onClick={async () =>
                                                await onEditASN(row)
                                            }
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
                </Grid>
            </Grid>
        </Formik>
    );
};
