import { useMemo } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import AddIcon from '@mui/icons-material/Add';
import EditIcon from '@mui/icons-material/Edit';
import FilterListIcon from '@mui/icons-material/FilterList';
import GavelIcon from '@mui/icons-material/Gavel';
import UndoIcon from '@mui/icons-material/Undo';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import IconButton from '@mui/material/IconButton';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { ColumnDef, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import { zodValidator } from '@tanstack/zod-form-adapter';
import { z } from 'zod';
import { apiGetBansGroups, BanReason, BanReasons, GroupBanRecord } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { DataTable } from '../component/DataTable.tsx';
import { Paginator } from '../component/Paginator.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { TableCellRelativeDateField } from '../component/TableCellRelativeDateField.tsx';
import { TableCellString } from '../component/TableCellString.tsx';
import { TableHeadingCell } from '../component/TableHeadingCell.tsx';
import { Buttons } from '../component/field/Buttons.tsx';
import { CheckboxSimple } from '../component/field/CheckboxSimple.tsx';
import { TextFieldSimple } from '../component/field/TextFieldSimple.tsx';
import { ModalBanGroup, ModalUnbanGroup } from '../component/modal';
import { commonTableSearchSchema, isPermanentBan, LazyResult, RowsPerPage } from '../util/table.ts';
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
    ...commonTableSearchSchema,
    sortColumn: z.enum(['ban_group_id', 'source_id', 'target_id', 'deleted', 'reason', 'valid_until']).optional(),
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
    const defaultRows = RowsPerPage.TwentyFive;
    const navigate = useNavigate({ from: Route.fullPath });
    const { page, rows, sortOrder, sortColumn, target_id, source_id, group_id, deleted } = Route.useSearch();
    const { data: bans, isLoading } = useQuery({
        queryKey: ['steamBans', { page, rows, sortOrder, sortColumn, target_id, source_id }],
        queryFn: async () => {
            return await apiGetBansGroups({
                limit: rows ?? defaultRows,
                offset: (page ?? 0) * (rows ?? defaultRows),
                order_by: sortColumn ?? 'ban_group_id',
                desc: (sortOrder ?? 'desc') == 'desc',
                source_id: source_id ?? '',
                target_id: target_id ?? '',
                group_id: group_id ?? ''
            });
        }
    });
    // const [newGroupBans, setNewGroupBans] = useState<GroupBanRecord[]>([]);
    // const { sendFlash } = useUserFlashCtx();

    const onNewGroup = async () => {
        await NiceModal.show(ModalBanGroup, {});
    };

    const onUnbanGroup = async (ban: GroupBanRecord) => {
        await NiceModal.show(ModalUnbanGroup, {
            banId: ban.ban_group_id
        });
    };

    const onEditGroup = async (ban: GroupBanRecord) => {
        await NiceModal.show(ModalBanGroup, {
            banId: ban.ban_group_id,
            existing: ban
        });
    };

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            await navigate({ to: '/admin/ban/group', search: (prev) => ({ ...prev, ...value }) });
        },
        validatorAdapter: zodValidator,
        validators: {
            onChange: banGroupSearchSchema
        },
        defaultValues: {
            source_id: source_id ?? '',
            target_id: target_id ?? '',
            group_id: group_id ?? '',
            deleted: deleted ?? false
        }
    });

    const clear = async () => {
        await navigate({
            to: '/admin/ban/group',
            search: (prev) => ({ ...prev, source_id: '', target_id: '', group_id: '', deleted: false })
        });
    };

    return (
        <Grid container spacing={2}>
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
                    {/*<BanGroupTable newBans={newGroupBans} />*/}
                    <BanGroupTable
                        bans={bans ?? { data: [], count: 0 }}
                        isLoading={isLoading}
                        onEditGroup={onEditGroup}
                        onUnbanGroup={onUnbanGroup}
                    />
                    <Paginator page={page ?? 0} rows={rows ?? defaultRows} data={bans} path={'/admin/ban/group'} />
                </ContainerWithHeaderAndButtons>
            </Grid>
        </Grid>
    );
}

const BanGroupTable = ({
    bans,
    isLoading,
    onEditGroup,
    onUnbanGroup
}: {
    bans: LazyResult<GroupBanRecord>;
    isLoading: boolean;
    onUnbanGroup: (ban: GroupBanRecord) => Promise<void>;
    onEditGroup: (ban: GroupBanRecord) => Promise<void>;
}) => {
    const columns = useMemo<ColumnDef<GroupBanRecord>[]>(
        () => [
            {
                accessorKey: 'ban_group_id',
                header: () => <TableHeadingCell name={'Ban ID'} />,
                cell: (info) => <TableCellString>{`#${info.getValue()}`}</TableCellString>
            },
            {
                accessorKey: 'source_id',
                header: () => <TableHeadingCell name={'Author'} />,
                cell: (info) => {
                    return typeof bans.data[info.row.index] === 'undefined' ? (
                        ''
                    ) : (
                        <PersonCell
                            steam_id={bans.data[info.row.index].source_id}
                            personaname={bans.data[info.row.index].source_personaname}
                            avatar_hash={bans.data[info.row.index].source_avatarhash}
                        />
                    );
                }
            },
            {
                accessorKey: 'target_id',
                header: () => <TableHeadingCell name={'Subject'} />,
                cell: (info) => {
                    return typeof bans.data[info.row.index] === 'undefined' ? (
                        ''
                    ) : (
                        <PersonCell
                            steam_id={bans.data[info.row.index].target_id}
                            personaname={bans.data[info.row.index].target_personaname}
                            avatar_hash={bans.data[info.row.index].target_avatarhash}
                        />
                    );
                }
            },
            {
                accessorKey: 'group_id',
                header: () => <TableHeadingCell name={'Group ID'} />,
                cell: (info) => <Typography>{`${info.getValue()}`}</Typography>
            },
            {
                accessorKey: 'reason',
                header: () => <TableHeadingCell name={'Reason'} />,
                cell: (info) => <Typography>{BanReasons[info.getValue() as BanReason]}</Typography>
            },
            {
                accessorKey: 'created_on',
                header: () => <TableHeadingCell name={'Created'} />,
                cell: (info) => <Typography>{renderDate(info.getValue() as Date)}</Typography>
            },
            {
                accessorKey: 'valid_until',
                header: () => <TableHeadingCell name={'Expires'} />,
                cell: (info) => {
                    return typeof bans.data[info.row.index] === 'undefined' ? (
                        ''
                    ) : isPermanentBan(bans.data[info.row.index].created_on, bans.data[info.row.index].valid_until) ? (
                        'Permanent'
                    ) : (
                        <TableCellRelativeDateField
                            date={bans.data[info.row.index].created_on}
                            compareDate={bans.data[info.row.index].valid_until}
                        />
                    );
                }
            },

            {
                id: 'actions',
                header: () => {
                    return <TableHeadingCell name={'Actions'} />;
                },
                cell: (info) => {
                    return (
                        <ButtonGroup fullWidth>
                            <IconButton
                                color={'warning'}
                                onClick={async () => {
                                    await onEditGroup(info.row.original);
                                }}
                            >
                                <Tooltip title={'Edit Ban'}>
                                    <EditIcon />
                                </Tooltip>
                            </IconButton>
                            <IconButton
                                color={'success'}
                                onClick={async () => {
                                    await onUnbanGroup(info.row.original);
                                }}
                            >
                                <Tooltip title={'Remove Ban'}>
                                    <UndoIcon />
                                </Tooltip>
                            </IconButton>
                        </ButtonGroup>
                    );
                }
            }
        ],
        [bans.data, onEditGroup, onUnbanGroup]
    );

    const table = useReactTable({
        data: bans.data,
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        manualPagination: true,
        autoResetPageIndex: true
    });

    return <DataTable table={table} isLoading={isLoading} />;
};
