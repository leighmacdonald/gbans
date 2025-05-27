import { useMemo, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import AddIcon from '@mui/icons-material/Add';
import EditIcon from '@mui/icons-material/Edit';
import FilterListIcon from '@mui/icons-material/FilterList';
import GavelIcon from '@mui/icons-material/Gavel';
import UndoIcon from '@mui/icons-material/Undo';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import Grid from '@mui/material/Grid';
import IconButton from '@mui/material/IconButton';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { ColumnFiltersState, createColumnHelper, PaginationState, SortingState } from '@tanstack/react-table';
import { z } from 'zod/v4';
import { apiGetBansGroups } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { Title } from '../component/Title';
import { ModalBanGroup, ModalUnbanGroup } from '../component/modal';
import { FullTable } from '../component/table/FullTable.tsx';
import { TableCellRelativeDateField } from '../component/table/TableCellRelativeDateField.tsx';
import { TableCellString } from '../component/table/TableCellString.tsx';
import { useAppForm } from '../contexts/formContext.tsx';
import { BanReasonEnum, BanReasons, GroupBanRecord } from '../schema/bans.ts';
import { commonTableSearchSchema, initColumnFilter, initPagination, isPermanentBan } from '../util/table.ts';
import { renderDate } from '../util/time.ts';
import { emptyOrNullString } from '../util/types.ts';

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

const searchSchema = commonTableSearchSchema.extend({
    sortColumn: z.enum(['ban_group_id', 'source_id', 'target_id', 'deleted', 'reason', 'valid_until']).optional(),
    source_id: sourceIDValidator,
    target_id: targetIDValidator,
    group_id: groupIDValidator,
    deleted: deletedValidator
});

export const Route = createFileRoute('/_mod/admin/ban/group')({
    component: AdminBanGroup,
    validateSearch: (search) => searchSchema.parse(search)
});

const schema = z.object({
    source_id: z.string(),
    target_id: z.string(),
    group_id: z.string().refine((arg) => {
        if (emptyOrNullString(arg)) {
            return true;
        }
        return arg?.match(/^\d+$/);
    }, 'Invalid group ID'),
    deleted: z.boolean()
});
function AdminBanGroup() {
    const navigate = useNavigate({ from: Route.fullPath });
    const search = Route.useSearch();
    const [pagination, setPagination] = useState<PaginationState>(initPagination(search.pageIndex, search.pageSize));
    const [sorting] = useState<SortingState>([{ id: 'ban_group_id', desc: true }]);
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

    const form = useAppForm({
        onSubmit: async ({ value }) => {
            setColumnFilters(initColumnFilter(value));
            await navigate({ to: '/admin/ban/group', search: (prev) => ({ ...prev, ...value }) });
        },
        validators: {
            onChange: schema
        },
        defaultValues: {
            source_id: search.source_id ?? '',
            target_id: search.target_id ?? '',
            group_id: search.group_id ?? '',
            deleted: search.deleted ?? false
        }
    });

    const clear = async () => {
        form.reset();
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
            <Grid size={{ xs: 12 }}>
                <ContainerWithHeader title={'Filters'} iconLeft={<FilterListIcon />}>
                    <form
                        onSubmit={async (e) => {
                            e.preventDefault();
                            e.stopPropagation();
                            await form.handleSubmit();
                        }}
                    >
                        <Grid container spacing={2}>
                            <Grid size={{ xs: 6, md: 3 }}>
                                <form.AppField
                                    name={'source_id'}
                                    children={(field) => {
                                        return <field.SteamIDField label={'Author Steam ID'} />;
                                    }}
                                />
                            </Grid>
                            <Grid size={{ xs: 6, md: 3 }}>
                                <form.AppField
                                    name={'target_id'}
                                    children={(field) => {
                                        return <field.SteamIDField label={'Subject Steam ID'} />;
                                    }}
                                />
                            </Grid>

                            <Grid size={{ xs: 6, md: 3 }}>
                                <form.AppField
                                    name={'group_id'}
                                    children={(field) => {
                                        return <field.TextField label={'Group ID'} />;
                                    }}
                                />
                            </Grid>

                            <Grid>
                                <form.AppField
                                    name={'deleted'}
                                    children={(field) => {
                                        return <field.CheckboxField label={'Show Deleted'} />;
                                    }}
                                />
                            </Grid>

                            <Grid size={{ xs: 12 }}>
                                <form.AppForm>
                                    <ButtonGroup>
                                        <form.ClearButton onClick={clear} />
                                        <form.ResetButton />
                                        <form.SubmitButton />
                                    </ButtonGroup>
                                </form.AppForm>
                            </Grid>
                        </Grid>
                    </form>
                </ContainerWithHeader>
            </Grid>
            <Grid size={{ xs: 12 }}>
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
                        toOptions={{ from: Route.fullPath }}
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
        header: 'Ban ID',
        size: 50,
        cell: (info) => <TableCellString>{`#${info.getValue()}`}</TableCellString>
    }),
    columnHelper.accessor('source_id', {
        header: 'Author',
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
        header: 'Subject',
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
        header: () => 'Group ID',
        cell: (info) => <Typography>{`${info.getValue()}`}</Typography>
    }),
    columnHelper.accessor('reason', {
        header: 'Reason',
        size: 100,
        cell: (info) => <Typography>{BanReasons[info.getValue() as BanReasonEnum]}</Typography>
    }),
    columnHelper.accessor('created_on', {
        header: 'Created',
        size: 100,
        cell: (info) => <Typography>{renderDate(info.getValue() as Date)}</Typography>
    }),
    columnHelper.accessor('valid_until', {
        size: 100,
        header: 'Expires',
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
        size: 30,

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
        size: 30,
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
