import React, { useCallback, useEffect, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import EditIcon from '@mui/icons-material/Edit';
import LinkIcon from '@mui/icons-material/Link';
import UndoIcon from '@mui/icons-material/Undo';
import ButtonGroup from '@mui/material/ButtonGroup';
import IconButton from '@mui/material/IconButton';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { formatDuration, intervalToDuration } from 'date-fns';
import { Formik } from 'formik';
import * as yup from 'yup';
import { apiGetBansGroups, BanGroupQueryFilter, GroupBanRecord } from '../api';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { logErr } from '../util/errors';
import { renderDate } from '../util/text';
import { LazyTable, Order, RowsPerPage } from './LazyTable';
import { TableCellBool } from './TableCellBool';
import { DeletedField, deletedValidator } from './formik/DeletedField';
import { FilterButtons } from './formik/FilterButtons';
import { GroupIdField, groupIdFieldValidator } from './formik/GroupIdField';
import { SourceIdField, sourceIdValidator } from './formik/SourceIdField';
import { SteamIDSelectField } from './formik/SteamIDSelectField';
import { TargetIDField, targetIdValidator } from './formik/TargetIdField';
import { ModalBanGroup, ModalUnbanGroup } from './modal';
import { BanGroupModalProps } from './modal/BanGroupModal';

interface GroupBanFilterValues {
    group_id: string;
    source_id: string;
    target_id: string;
    deleted: boolean;
}

const validationSchema = yup.object({
    group_id: groupIdFieldValidator,
    source_id: sourceIdValidator,
    target_id: targetIdValidator,
    deleted: deletedValidator
});

export const BanGroupTable = () => {
    const [bans, setBans] = useState<GroupBanRecord[]>([]);
    const [sortOrder, setSortOrder] = useState<Order>('desc');
    const [sortColumn, setSortColumn] =
        useState<keyof GroupBanRecord>('ban_group_id');
    const [rowPerPageCount, setRowPerPageCount] = useState<number>(
        RowsPerPage.TwentyFive
    );
    const [page, setPage] = useState(0);
    const [totalRows, setTotalRows] = useState<number>(0);
    const [source, setSource] = useState('');
    const [group, setGroup] = useState('');
    const [target, setTarget] = useState('');
    const [deleted, setDeleted] = useState(false);
    const { sendFlash } = useUserFlashCtx();

    const onEditGroup = useCallback(
        async (existing: GroupBanRecord) => {
            try {
                await NiceModal.show<GroupBanRecord, BanGroupModalProps>(
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
        const opts: BanGroupQueryFilter = {
            limit: rowPerPageCount,
            offset: page * rowPerPageCount,
            order_by: sortColumn,
            desc: sortOrder == 'desc',
            deleted: deleted,
            source_id: source,
            target_id: target,
            group_id: group
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
    }, [
        deleted,
        group,
        page,
        rowPerPageCount,
        sortColumn,
        sortOrder,
        source,
        target
    ]);

    const iv: GroupBanFilterValues = {
        group_id: '',
        source_id: '',
        target_id: '',
        deleted: false
    };

    const onSubmit = useCallback((values: GroupBanFilterValues) => {
        setGroup(values.group_id);
        setSource(values.source_id);
        setTarget(values.target_id);
        setDeleted(values.deleted);
    }, []);

    const onReset = useCallback(() => {
        setGroup(iv.group_id);
        setSource(iv.source_id);
        setTarget(iv.target_id);
        setDeleted(iv.deleted);
    }, [iv.group_id, iv.source_id, iv.deleted, iv.target_id]);

    return (
        <Formik<GroupBanFilterValues>
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
                            <GroupIdField />
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
                    <LazyTable<GroupBanRecord>
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
                                sortKey: 'ban_group_id',
                                sortable: true,
                                align: 'left',
                                renderer: (obj) => (
                                    <Typography variant={'body1'}>
                                        #{obj.ban_group_id.toString()}
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
                                label: 'GroupID',
                                tooltip: 'GroupID',
                                sortKey: 'group_id',
                                sortable: true,
                                align: 'left',
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
                                renderer: (row) => (
                                    <Typography variant={'body1'}>
                                        {row.note}
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
                                            color={'info'}
                                            onClick={() => {
                                                window.open(
                                                    `https://steamcommunity.com/gid/${row.group_id}`,
                                                    '_blank',
                                                    'noreferrer'
                                                );
                                            }}
                                        >
                                            <Tooltip title={'Open Steam Group'}>
                                                <LinkIcon />
                                            </Tooltip>
                                        </IconButton>
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
                                                await onUnbanGroup(
                                                    row.ban_group_id
                                                );
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
