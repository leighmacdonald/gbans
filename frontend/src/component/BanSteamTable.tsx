import React, { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import NiceModal from '@ebay/nice-modal-react';
import EditIcon from '@mui/icons-material/Edit';
import UndoIcon from '@mui/icons-material/Undo';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import format from 'date-fns/format';
import { Formik } from 'formik';
import * as yup from 'yup';
import {
    apiGetBansSteam,
    AppealState,
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
import {
    AppealStateField,
    appealStateFielValidator
} from './formik/AppealStateField';
import { AuthorIDField, authorIdValidator } from './formik/AuthorIdField';
import { DeletedField, deletedValidator } from './formik/DeletedField';
import { FilterButtons } from './formik/FilterButtons';
import { TargetIDField, targetIdValidator } from './formik/TargetIdField';
import { ModalBanSteam, ModalUnbanSteam } from './modal';

interface SteamBanFilterValues {
    appeal_state: AppealState;
    author_id: string;
    target_id: string;
    deleted: boolean;
}

const validationSchema = yup.object({
    appeal_state: appealStateFielValidator,
    author_id: authorIdValidator,
    target_id: targetIdValidator,
    deleted: deletedValidator
});

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
    const [author, setAuthor] = useState('');
    const [target, setTarget] = useState('');
    const [deleted, setDeleted] = useState(false);
    const [appealState, setAppealState] = useState<AppealState>(
        AppealState.Any
    );
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
            desc: sortOrder == 'desc',
            source_id: author,
            target_id: target,
            appeal_state: appealState,
            deleted: deleted
        };

        apiGetBansSteam(opts, abortController)
            .then((bans) => {
                setBans(bans.data);
                setTotalRows(bans.count);
                if (page * rowPerPageCount > bans.count) {
                    setPage(0);
                }
            })
            .catch((reason) => {
                logErr(reason);
            });

        return () => abortController.abort();
    }, [
        appealState,
        author,
        deleted,
        page,
        rowPerPageCount,
        sortColumn,
        sortOrder,
        target
    ]);

    const iv: SteamBanFilterValues = {
        appeal_state: AppealState.Any,
        author_id: '',
        target_id: '',
        deleted: false
    };

    const onSubmit = useCallback((values: SteamBanFilterValues) => {
        setAppealState(values.appeal_state);
        setAuthor(values.author_id);
        setTarget(values.target_id);
        setDeleted(values.deleted);
    }, []);

    const onReset = useCallback(() => {
        setAppealState(iv.appeal_state);
        setAuthor(iv.author_id);
        setTarget(iv.target_id);
        setDeleted(iv.deleted);
    }, [iv.appeal_state, iv.author_id, iv.deleted, iv.target_id]);

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
                    <Grid container>
                        <Grid xs={12} padding={2}>
                            <Stack direction={'row'} spacing={2}>
                                <AppealStateField />
                                <AuthorIDField />
                                <TargetIDField />
                                <DeletedField />
                                <FilterButtons />
                            </Stack>
                        </Grid>
                    </Grid>
                </Grid>
                <Grid xs={12}>
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
                                    <PersonCell
                                        onClick={() => {
                                            setAuthor(row.source_id);
                                        }}
                                        steam_id={row.source_id}
                                        personaname={''}
                                        avatar_hash={row.source_avatarhash}
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
                                    <PersonCell
                                        onClick={() => {
                                            setTarget(row.target_id);
                                        }}
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
                                                    navigate(
                                                        `/report/${row.report_id}`
                                                    );
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
                </Grid>
            </Grid>
        </Formik>
    );
};