import { useMemo, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import AddIcon from '@mui/icons-material/Add';
import EditIcon from '@mui/icons-material/Edit';
import FilterListIcon from '@mui/icons-material/FilterList';
import GavelIcon from '@mui/icons-material/Gavel';
import UndoIcon from '@mui/icons-material/Undo';
import Button from '@mui/material/Button';
import IconButton from '@mui/material/IconButton';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { ColumnFiltersState, createColumnHelper, PaginationState, SortingState } from '@tanstack/react-table';
import { zodValidator } from '@tanstack/zod-form-adapter';
import { z } from 'zod';
import { apiGetBansGroups, BanReason, BanReasons, GroupBanRecord } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { FullTable } from '../component/FullTable.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { TableCellRelativeDateField } from '../component/TableCellRelativeDateField.tsx';
import { TableCellString } from '../component/TableCellString.tsx';
import { TableHeadingCell } from '../component/TableHeadingCell.tsx';
import { Title } from '../component/Title';
import { Buttons } from '../component/field/Buttons.tsx';
import { CheckboxSimple } from '../component/field/CheckboxSimple.tsx';
import { TextFieldSimple } from '../component/field/TextFieldSimple.tsx';
import { ModalBanGroup, ModalUnbanGroup } from '../component/modal';
import { initColumnFilter, initPagination, isPermanentBan, makeCommonTableSearchSchema } from '../util/table.ts';
import { renderDate } from '../util/text.tsx';
import { emptyOrNullString } from '../util/types.ts';
import { makeSteamidValidatorsOptional } from '../util/validator/makeSteamidValidatorsOptional.ts';

const sourceIDValidator = z.string().optional();
const targetIDValidator = z.string().optional();
const groupIDValidator = z
    .string()
    .optional()
    .refine((arg) => {
        if (emptyOrNullString(arg)) {
            return true;
        }
        return arg?.match(/^\d+$/);
    }, 'Invalid group ID');
const deletedValidator = z.boolean().optional();

const banGroupSearchSchema = z.object({
    ...makeCommonTableSearchSchema(['ban_group_id', 'source_id', 'target_id', 'deleted', 'reason', 'valid_until']),
    source_id: sourceIDValidator,
    target_id: targetIDValidator,
    group_id: groupIDValidator,
    deleted: deletedValidator
});

export const Route = createFileRoute('/_mod/admin/ban/group')({
    component: AdminBanGroup,
    validateSearch: (search) => banGroupSearchSchema.parse(search)
});

function AdminBanGroup() {
    const navigate = useNavigate({ from: Route.fullPath });
    const search = Route.useSearch();
    const [pagination, setPagination] = useState<PaginationState>(initPagination(search.pageIndex, search.pageSize));
    const [sorting] = useState<SortingState>([{ id: 'ban_id', desc: true }]);
    const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>(initColumnFilter(search));

    const { data: bans, isLoading } = useQuery({
        queryKey: ['steamGroupBans'],
        queryFn: async () => {
            return await apiGetBansGroups({ deleted: false });
        }
    });

    const onNewGroup = async () => {
        await NiceModal.show(ModalBanGroup, {});
    };

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            setColumnFilters(initColumnFilter(value));
            await navigate({ to: '/admin/ban/group', search: (prev) => ({ ...prev, ...value }) });
        },
        validatorAdapter: zodValidator,
        validators: {
            onChange: banGroupSearchSchema
        },
        defaultValues: {
            source_id: search.source_id ?? '',
            target_id: search.target_id ?? '',
            group_id: search.group_id ?? '',
            deleted: search.deleted ?? false
        }
    });

    const clear = async () => {
        reset();
        setColumnFilters([]);
        await navigate({
            to: '/admin/ban/group',
            search: (prev) => ({ ...prev, source_id: '', target_id: '', group_id: '', deleted: false })
        });
    };

    const columns = useMemo(() => {
        const onUnban = async (ban: GroupBanRecord) => {
            await NiceModal.show(ModalUnbanGroup, {
                banId: ban.ban_group_id
            });
        };

        const onEdit = async (ban: GroupBanRecord) => {
            await NiceModal.show(ModalBanGroup, {
                banId: ban.ban_group_id,
                existing: ban
            });
        };
        return makeColumns(onEdit, onUnban);
    }, []);

    return (
        <Grid container spacing={2}>
            <Title>Ban Group</Title>
            <Grid xs={12}>
                <ContainerWithHeader title={'Filters'} iconLeft={<FilterListIcon />}>
                    <form
                        onSubmit={async (e) => {
                            e.preventDefault();
                            e.stopPropagation();
                            await handleSubmit();
                        }}
                    >
                        <Grid container spacing={2}>
                            <Grid xs={4}>
                                <Grid xs={6} md={3}>
                                    <Field
                                        name={'source_id'}
                                        validators={makeSteamidValidatorsOptional()}
                                        children={(props) => {
                                            return (
                                                <TextFieldSimple
                                                    {...props}
                                                    label={'Author Steam ID'}
                                                    fullwidth={true}
                                                />
                                            );
                                        }}
                                    />
                                </Grid>
                            </Grid>
                            <Grid xs={6} md={3}>
                                <Field
                                    name={'target_id'}
                                    validators={makeSteamidValidatorsOptional()}
                                    children={(props) => {
                                        return (
                                            <TextFieldSimple {...props} label={'Subject Steam ID'} fullwidth={true} />
                                        );
                                    }}
                                />
                            </Grid>

                            <Grid xs={6} md={3}>
                                <Field
                                    name={'group_id'}
                                    validators={{
                                        onChange: groupIDValidator
                                    }}
                                    children={(props) => {
                                        return <TextFieldSimple {...props} label={'Group ID'} fullwidth={true} />;
                                    }}
                                />
                            </Grid>

                            <Grid xs="auto">
                                <Field
                                    name={'deleted'}
                                    children={(props) => {
                                        return <CheckboxSimple {...props} label={'Show Deleted'} />;
                                    }}
                                />
                            </Grid>

                            <Grid xs={12} mdOffset="auto">
                                <Subscribe
                                    selector={(state) => [state.canSubmit, state.isSubmitting]}
                                    children={([canSubmit, isSubmitting]) => (
                                        <Buttons
                                            reset={reset}
                                            canSubmit={canSubmit}
                                            isSubmitting={isSubmitting}
                                            onClear={clear}
                                        />
                                    )}
                                />
                            </Grid>
                        </Grid>
                    </form>
                </ContainerWithHeader>
            </Grid>
            <Grid xs={12}>
                <ContainerWithHeaderAndButtons
                    title={'Steam Group Ban History'}
                    marginTop={0}
                    iconLeft={<GavelIcon />}
                    buttons={[
                        <Button
                            key={`ban-group`}
                            variant={'contained'}
                            color={'success'}
                            startIcon={<AddIcon />}
                            sx={{ marginRight: 2 }}
                            onClick={onNewGroup}
                        >
                            Create
                        </Button>
                    ]}
                >
                    <FullTable
                        columnFilters={columnFilters}
                        pagination={pagination}
                        setPagination={setPagination}
                        data={bans ?? []}
                        isLoading={isLoading}
                        columns={columns}
                        sorting={sorting}
                    />
                </ContainerWithHeaderAndButtons>
            </Grid>
        </Grid>
    );
}

const columnHelper = createColumnHelper<GroupBanRecord>();

const makeColumns = (
    onEdit: (ban: GroupBanRecord) => Promise<void>,
    onUnban: (ban: GroupBanRecord) => Promise<void>
) => [
    columnHelper.accessor('ban_group_id', {
        header: () => <TableHeadingCell name={'Ban ID'} />,
        cell: (info) => <TableCellString>{`#${info.getValue()}`}</TableCellString>
    }),
    columnHelper.accessor('source_id', {
        header: () => <TableHeadingCell name={'Author'} />,
        cell: (info) => {
            return typeof info.row.original === 'undefined' ? (
                ''
            ) : (
                <PersonCell
                    showCopy={true}
                    steam_id={info.row.original.source_id}
                    personaname={info.row.original.source_personaname}
                    avatar_hash={info.row.original.source_avatarhash}
                />
            );
        }
    }),
    columnHelper.accessor('target_id', {
        header: () => <TableHeadingCell name={'Subject'} />,
        cell: (info) => {
            return typeof info.row.original === 'undefined' ? (
                ''
            ) : (
                <PersonCell
                    steam_id={info.row.original.target_id}
                    personaname={info.row.original.target_personaname}
                    avatar_hash={info.row.original.target_avatarhash}
                />
            );
        }
    }),
    columnHelper.accessor('group_id', {
        header: () => <TableHeadingCell name={'Group ID'} />,
        cell: (info) => <Typography>{`${info.getValue()}`}</Typography>
    }),
    columnHelper.accessor('reason', {
        header: () => <TableHeadingCell name={'Reason'} />,
        cell: (info) => <Typography>{BanReasons[info.getValue() as BanReason]}</Typography>
    }),
    columnHelper.accessor('created_on', {
        header: () => <TableHeadingCell name={'Created'} />,
        cell: (info) => <Typography>{renderDate(info.getValue() as Date)}</Typography>
    }),
    columnHelper.accessor('valid_until', {
        header: () => <TableHeadingCell name={'Expires'} />,
        cell: (info) => {
            return typeof info.row.original === 'undefined' ? (
                ''
            ) : isPermanentBan(info.row.original.created_on, info.row.original.valid_until) ? (
                'Permanent'
            ) : (
                <TableCellRelativeDateField
                    date={info.row.original.created_on}
                    compareDate={info.row.original.valid_until}
                />
            );
        }
    }),
    columnHelper.display({
        id: 'edit',
        header: () => {
            return <TableHeadingCell name={'Actions'} />;
        },
        cell: (info) => {
            return (
                <IconButton
                    color={'warning'}
                    onClick={async () => {
                        await onEdit(info.row.original);
                    }}
                >
                    <Tooltip title={'Edit Ban'}>
                        <EditIcon />
                    </Tooltip>
                </IconButton>
            );
        }
    }),
    columnHelper.display({
        id: 'unban',
        header: () => {
            return <TableHeadingCell name={'Actions'} />;
        },
        cell: (info) => {
            return (
                <IconButton
                    color={'success'}
                    onClick={async () => {
                        await onUnban(info.row.original);
                    }}
                >
                    <Tooltip title={'Remove Ban'}>
                        <UndoIcon />
                    </Tooltip>
                </IconButton>
            );
        }
    })
];
