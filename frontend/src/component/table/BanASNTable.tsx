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
import * as yup from 'yup';
import { ASNBanRecord, BanReason } from '../../api';
import { useUserFlashCtx } from '../../contexts/UserFlashCtx';
import { useBansASN } from '../../hooks/useBansASN';
import { renderDate } from '../../util/text';
import { ASNumberField, asNumberFieldValidator } from '../formik/ASNumberField';
import { DeletedField, deletedValidator } from '../formik/DeletedField';
import { FilterButtons } from '../formik/FilterButtons';
import { SourceIdField, sourceIdValidator } from '../formik/SourceIdField';
import { SteamIDSelectField } from '../formik/SteamIDSelectField';
import { TargetIDField, targetIdValidator } from '../formik/TargetIdField';
import { ModalBanASN, ModalUnbanASN } from '../modal';
import { BanASNModalProps } from '../modal/BanASNModal';
import { LazyTable, RowsPerPage } from './LazyTable';
import { TableCellBool } from './TableCellBool';

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

export const BanASNTable = ({ newBans }: { newBans: ASNBanRecord[] }) => {
    const [state, setState] = useUrlState({
        page: undefined,
        source: undefined,
        target: undefined,
        deleted: undefined,
        asNum: undefined,
        rows: undefined,
        sortOrder: undefined,
        sortColumn: undefined
    });
    const { sendFlash } = useUserFlashCtx();

    const { data, count } = useBansASN({
        limit: Number(state.rows ?? RowsPerPage.Ten),
        offset: Number((state.page ?? 0) * (state.rows ?? RowsPerPage.Ten)),
        order_by: state.sortColumn ?? 'ban_id',
        desc: (state.sortOrder ?? 'desc') == 'desc',
        source_id: state.source ?? '',
        target_id: state.target ?? '',
        as_num: Number(state.asNum ?? 0),
        deleted: state.deleted != '' ? Boolean(state.deleted) : false
    });

    const allBans = useMemo(() => {
        if (newBans.length > 0) {
            return [...newBans, ...data];
        }

        return data;
    }, [data, newBans]);

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

    const onSubmit = useCallback(
        (values: ASNFilterValues) => {
            const newState = {
                asNum: values.as_num != 0 ? values.as_num : undefined,
                source: values.source_id != '' ? values.source_id : undefined,
                target: values.target_id != '' ? values.target_id : undefined,
                deleted: values.deleted ? true : undefined
            };
            setState(newState);
        },
        [setState]
    );

    const onReset = useCallback(() => {
        setState({
            asNum: undefined,
            source: undefined,
            target: undefined,
            deleted: undefined
        });
    }, [setState]);

    return (
        <Formik
            initialValues={{
                as_num: Number(state.asNum),
                source_id: state.source,
                target_id: state.target,
                deleted: Boolean(state.deleted)
            }}
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
                                    <SteamIDSelectField
                                        steam_id={row.source_id}
                                        personaname={row.source_personaname}
                                        avatarhash={row.source_avatarhash}
                                        field_name={'source_id'}
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
                                    <SteamIDSelectField
                                        steam_id={row.target_id}
                                        personaname={row.target_personaname}
                                        avatarhash={row.target_avatarhash}
                                        field_name={'target_id'}
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
                                        <Typography variant={'body1'}>
                                            {renderDate(obj.created_on)}
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
                                            {renderDate(obj.valid_until)}
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
