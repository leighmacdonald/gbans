import { ChangeEvent, useCallback, useMemo } from 'react';
import useUrlState from '@ahooksjs/use-url-state';
import NiceModal from '@ebay/nice-modal-react';
import EditIcon from '@mui/icons-material/Edit';
import InfoIcon from '@mui/icons-material/Info';
import LinkIcon from '@mui/icons-material/Link';
import UndoIcon from '@mui/icons-material/Undo';
import ButtonGroup from '@mui/material/ButtonGroup';
import IconButton from '@mui/material/IconButton';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { formatDuration, intervalToDuration } from 'date-fns';
import { Formik } from 'formik';
import { FormikHelpers } from 'formik/dist/types';
import * as yup from 'yup';
import { GroupBanRecord } from '../../api';
import { useUserFlashCtx } from '../../contexts/UserFlashCtx';
import { useBansGroup } from '../../hooks/useBansGroup';
import { renderDate } from '../../util/text';
import { DeletedField, deletedValidator } from '../formik/DeletedField';
import { FilterButtons } from '../formik/FilterButtons';
import { GroupIdField, groupIdFieldValidator } from '../formik/GroupIdField';
import { SourceIdField, sourceIdValidator } from '../formik/SourceIdField';
import { SteamIDSelectField } from '../formik/SteamIDSelectField';
import { TargetIDField, targetIdValidator } from '../formik/TargetIdField';
import { ModalBanGroup, ModalUnbanGroup } from '../modal';
import { BanGroupModalProps } from '../modal/BanGroupModal';
import { LazyTable, RowsPerPage } from './LazyTable';
import { TableCellBool } from './TableCellBool';

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

export const BanGroupTable = ({ newBans }: { newBans: GroupBanRecord[] }) => {
    const [state, setState] = useUrlState({
        page: undefined,
        source: undefined,
        target: undefined,
        deleted: undefined,
        group: undefined,
        rows: undefined,
        sortOrder: undefined,
        sortColumn: undefined
    });
    const { sendFlash } = useUserFlashCtx();

    const { data, count } = useBansGroup({
        limit: Number(state.rows ?? RowsPerPage.Ten),
        offset: Number((state.page ?? 0) * (state.rows ?? RowsPerPage.Ten)),
        order_by: state.sortColumn ?? 'ban_group_id',
        desc: (state.sortOrder ?? 'desc') == 'desc',
        source_id: state.source ?? '',
        target_id: state.target ?? '',
        group_id: state.group ?? '',
        deleted: state.deleted != '' ? Boolean(state.deleted) : false
    });

    const allBans = useMemo(() => {
        if (newBans.length > 0) {
            return [...newBans, ...data];
        }

        return data;
    }, [data, newBans]);

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

    const onSubmit = useCallback(
        (values: GroupBanFilterValues) => {
            const newState = {
                group: values.group_id != '' ? values.group_id : undefined,
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
            _: GroupBanFilterValues,
            formikHelpers: FormikHelpers<GroupBanFilterValues>
        ) => {
            setState({
                group: undefined,
                source: undefined,
                target: undefined,
                deleted: undefined
            });
            await formikHelpers.setFieldValue('source_id', undefined);
            await formikHelpers.setFieldValue('target_id', undefined);
        },
        [setState]
    );

    return (
        <Formik<GroupBanFilterValues>
            initialValues={{
                group_id: '',
                source_id: '',
                target_id: '',
                deleted: false
            }}
            onReset={onReset}
            onSubmit={onSubmit}
            validationSchema={validationSchema}
            validateOnChange={true}
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
                            <GroupIdField />
                        </Grid>
                        <Grid xs={4} sm={3} md={2}>
                            <DeletedField />
                        </Grid>
                        <Grid xs={4} sm={3} md={2}>
                            <FilterButtons />
                        </Grid>
                    </Grid>
                </Grid>
                <Grid xs={12}>
                    <LazyTable<GroupBanRecord>
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
                            event: ChangeEvent<
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
                                align: 'center',
                                renderer: (row) => (
                                    <Tooltip title={row.note}>
                                        <InfoIcon />
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
