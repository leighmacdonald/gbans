import React, { useCallback, useMemo } from 'react';
import useUrlState from '@ahooksjs/use-url-state';
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
import { FormikHelpers } from 'formik/dist/types';
import IPCIDR from 'ip-cidr';
import * as yup from 'yup';
import { BanReason, CIDRBanRecord } from '../../api';
import { useUserFlashCtx } from '../../contexts/UserFlashCtx';
import { useBansCIDR } from '../../hooks/useBansCIDR';
import { logErr } from '../../util/errors';
import { DeletedField, deletedValidator } from '../formik/DeletedField';
import { FilterButtons } from '../formik/FilterButtons';
import { IPField, ipFieldValidator } from '../formik/IPField';
import { SourceIdField, sourceIdValidator } from '../formik/SourceIdField';
import { SteamIDSelectField } from '../formik/SteamIDSelectField';
import { TargetIDField, targetIdValidator } from '../formik/TargetIdField';
import { ModalBanCIDR, ModalUnbanCIDR } from '../modal';
import { BanCIDRModalProps } from '../modal/BanCIDRModal';
import { LazyTable, RowsPerPage } from './LazyTable';
import { TableCellBool } from './TableCellBool';
import { TableCellRelativeDateField } from './TableCellRelativeDateField';

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

export const BanCIDRTable = ({ newBans }: { newBans: CIDRBanRecord[] }) => {
    const [state, setState] = useUrlState({
        page: undefined,
        source: undefined,
        target: undefined,
        deleted: undefined,
        ip: undefined,
        rows: undefined,
        sortOrder: undefined,
        sortColumn: undefined
    });
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
    const { data, count } = useBansCIDR({
        limit: Number(state.rows ?? RowsPerPage.Ten),
        offset: Number((state.page ?? 0) * (state.rows ?? RowsPerPage.Ten)),
        order_by: state.sortColumn ?? 'net_id',
        desc: (state.sortOrder ?? 'desc') == 'desc',
        source_id: state.source ?? '',
        target_id: state.target ?? '',
        ip: state.ip ?? '',
        deleted: state.deleted != '' ? Boolean(state.deleted) : false
    });

    const allBans = useMemo(() => {
        if (newBans.length > 0) {
            return [...newBans, ...data];
        }

        return data;
    }, [data, newBans]);

    const onSubmit = useCallback(
        (values: CIDRBanFilterValues) => {
            const newState = {
                ip: values.ip != '' ? values.ip : undefined,
                source: values.source_id != '' ? values.source_id : undefined,
                target: values.target_id != '' ? values.target_id : undefined,
                deleted: values.deleted ? true : undefined
            };
            setState(newState);
        },
        [setState]
    );

    const onReset = useCallback(
        async (
            _: CIDRBanFilterValues,
            formikHelpers: FormikHelpers<CIDRBanFilterValues>
        ) => {
            setState({
                ip: undefined,
                source: undefined,
                target: undefined,
                deleted: undefined
            });
            await formikHelpers.setFieldValue('source_id', '');
            await formikHelpers.setFieldValue('target_id', '');
            await formikHelpers.setFieldValue('ip', '');
        },
        [setState]
    );

    return (
        <Formik
            onSubmit={onSubmit}
            initialValues={{
                ip: state.ip,
                source_id: state.source,
                target_id: state.target,
                deleted: Boolean(state.deleted)
            }}
            onReset={onReset}
            validationSchema={validationSchema}
            validateOnChange={true}
            validateOnBlur={true}
        >
            <Grid container spacing={3}>
                <Grid xs={12}>
                    <Grid container spacing={2}>
                        <Grid xs={4} sm={3} md={2}>
                            <SourceIdField />
                        </Grid>
                        <Grid xs={4} sm={3} md={2}>
                            <TargetIDField />
                        </Grid>
                        <Grid xs={4} sm={3} md={2}>
                            <IPField />
                        </Grid>
                        <Grid xs={4} sm={3} md={2}>
                            <DeletedField />
                        </Grid>
                        <Grid xs={4} sm={3} md={4}>
                            <FilterButtons />
                        </Grid>
                    </Grid>
                </Grid>
                <Grid xs={12}>
                    <LazyTable<CIDRBanRecord>
                        showPager={true}
                        count={count}
                        rows={allBans}
                        page={Number(state.page ?? 0)}
                        rowsPerPage={Number(state.rows ?? RowsPerPage.Ten)}
                        sortOrder={state.sortOrder}
                        sortColumn={state.sortColumn}
                        onSortColumnChanged={async (column) => {
                            setState({ sortColumn: column });
                        }}
                        onSortOrderChanged={async (direction) => {
                            setState({ sortOrder: direction });
                        }}
                        onPageChange={(_, newPage: number) => {
                            setState({ page: newPage });
                        }}
                        onRowsPerPageChange={(
                            event: React.ChangeEvent<
                                HTMLInputElement | HTMLTextAreaElement
                            >
                        ) => {
                            setState({
                                rows: Number(event.target.value),
                                page: 0
                            });
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
                                    <SteamIDSelectField
                                        steam_id={row.source_id}
                                        personaname={row.source_personaname}
                                        avatarhash={row.source_avatarhash}
                                        field_name={'source_id'}
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
                                    <SteamIDSelectField
                                        steam_id={row.target_id}
                                        personaname={row.target_personaname}
                                        avatarhash={row.target_avatarhash}
                                        field_name={'target_id'}
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
                                    <Tooltip
                                        title={
                                            row.reason == BanReason.Custom
                                                ? row.reason_text
                                                : BanReason[row.reason]
                                        }
                                    >
                                        <Typography variant={'body1'}>
                                            {BanReason[row.reason]}
                                        </Typography>
                                    </Tooltip>
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
                                        <TableCellRelativeDateField
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
                                        <TableCellRelativeDateField
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
