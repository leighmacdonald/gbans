import React, { useCallback, useMemo, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import EditIcon from '@mui/icons-material/Edit';
import UndoIcon from '@mui/icons-material/Undo';
import Box from '@mui/material/Box';
import ButtonGroup from '@mui/material/ButtonGroup';
import IconButton from '@mui/material/IconButton';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { Formik } from 'formik';
import * as yup from 'yup';
import { AppealState, BanReason, SteamBanRecord } from '../../api';
import { useUserFlashCtx } from '../../contexts/UserFlashCtx';
import { useBansSteam } from '../../hooks/useBansSteam';
import { renderDate } from '../../util/text';
import {
    AppealStateField,
    appealStateFielValidator
} from '../formik/AppealStateField';
import { DeletedField, deletedValidator } from '../formik/DeletedField';
import { FilterButtons } from '../formik/FilterButtons';
import { SourceIdField, sourceIdValidator } from '../formik/SourceIdField';
import { SteamIDSelectField } from '../formik/SteamIDSelectField';
import { TargetIDField, targetIdValidator } from '../formik/TargetIdField';
import { ModalBanSteam, ModalUnbanSteam } from '../modal';
import { LazyTable, Order, RowsPerPage } from './LazyTable';
import { TableCellBool } from './TableCellBool';
import { TableCellLink } from './TableCellLink';
import {
    isPermanentBan,
    TableCellRelativeDateField
} from './TableCellRelativeDateField';

interface SteamBanFilterValues {
    appeal_state: AppealState;
    source_id: string;
    target_id: string;
    deleted: boolean;
}

const validationSchema = yup.object({
    appeal_state: appealStateFielValidator,
    source_id: sourceIdValidator,
    target_id: targetIdValidator,
    deleted: deletedValidator
});

export const BanSteamTable = ({ newBans }: { newBans: SteamBanRecord[] }) => {
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] =
        useState<keyof SteamBanRecord>('ban_id');
    const [rowPerPageCount, setRowPerPageCount] = useState<number>(
        RowsPerPage.TwentyFive
    );
    const [page, setPage] = useState(0);
    const [source, setSource] = useState('');
    const [target, setTarget] = useState('');
    const [deleted, setDeleted] = useState(false);
    const [appealState, setAppealState] = useState<AppealState>(
        AppealState.Any
    );
    const { sendFlash } = useUserFlashCtx();

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

    const { data, count } = useBansSteam({
        limit: rowPerPageCount,
        offset: page * rowPerPageCount,
        order_by: sortColumn,
        desc: sortOrder == 'desc',
        source_id: source,
        target_id: target,
        appeal_state: appealState,
        deleted: deleted
    });

    const allBans = useMemo(() => {
        if (newBans.length > 0) {
            return [...newBans, ...data];
        }

        return data;
    }, [data, newBans]);

    const iv: SteamBanFilterValues = {
        appeal_state: AppealState.Any,
        source_id: '',
        target_id: '',
        deleted: false
    };

    const onSubmit = useCallback((values: SteamBanFilterValues) => {
        setAppealState(values.appeal_state);
        setSource(values.source_id);
        setTarget(values.target_id);
        setDeleted(values.deleted);
    }, []);

    const onReset = useCallback(() => {
        setAppealState(iv.appeal_state);
        setSource(iv.source_id);
        setTarget(iv.target_id);
        setDeleted(iv.deleted);
    }, [iv.appeal_state, iv.source_id, iv.deleted, iv.target_id]);

    return (
        <Formik<SteamBanFilterValues>
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
                            <SourceIdField />
                        </Grid>
                        <Grid xs>
                            <TargetIDField />
                        </Grid>
                        <Grid xs>
                            <AppealStateField />
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
                    <LazyTable<SteamBanRecord>
                        showPager={true}
                        count={count}
                        rows={allBans}
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
                                label: 'Reason',
                                tooltip: 'Reason',
                                sortKey: 'reason',
                                sortable: true,
                                align: 'left',
                                renderer: (row) => (
                                    <Box>
                                        <Tooltip
                                            title={
                                                row.reason == BanReason.Custom
                                                    ? row.reason_text
                                                    : BanReason[row.reason]
                                            }
                                        >
                                            <Typography variant={'body1'}>
                                                {`${BanReason[row.reason]}`}
                                            </Typography>
                                        </Tooltip>
                                    </Box>
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
                                    return isPermanentBan(
                                        row.created_on,
                                        row.valid_until
                                    ) ? (
                                        'Permanent'
                                    ) : (
                                        <TableCellRelativeDateField
                                            date={row.created_on}
                                            compareDate={row.valid_until}
                                        />
                                    );
                                }
                            },
                            {
                                label: 'F',
                                tooltip: 'Are friends also included in the ban',
                                align: 'center',
                                width: '50px',
                                sortKey: 'include_friends',
                                renderer: (row) => (
                                    <TableCellBool
                                        enabled={row.include_friends}
                                    />
                                )
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
                                label: 'Rep.',
                                tooltip: 'Report',
                                sortable: false,
                                align: 'center',
                                width: '20px',
                                renderer: (row) =>
                                    row.report_id > 0 ? (
                                        <Tooltip title={'View Report'}>
                                            <>
                                                <TableCellLink
                                                    label={`#${row.report_id}`}
                                                    to={`/report/${row.report_id}`}
                                                />
                                            </>
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
                                align: 'center',
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
                </Grid>
            </Grid>
        </Formik>
    );
};
