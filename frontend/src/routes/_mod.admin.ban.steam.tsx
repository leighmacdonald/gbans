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
import Link from '@mui/material/Link';
import MenuItem from '@mui/material/MenuItem';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { ColumnDef, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import { z } from 'zod';
import {
    apiGetBansSteam,
    AppealState,
    AppealStateCollection,
    appealStateString,
    BanReason,
    BanReasons,
    PermissionLevel,
    SteamBanRecord
} from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { DataTable } from '../component/DataTable.tsx';
import { Paginator } from '../component/Paginator.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import RouterLink from '../component/RouterLink.tsx';
import { TableCellBool } from '../component/TableCellBool.tsx';
import { TableCellRelativeDateField } from '../component/TableCellRelativeDateField.tsx';
import { TableHeadingCell } from '../component/TableHeadingCell.tsx';
import { Buttons } from '../component/field/Buttons.tsx';
import { CheckboxSimple } from '../component/field/CheckboxSimple.tsx';
import { SelectFieldSimple } from '../component/field/SelectFieldSimple.tsx';
import { TextFieldSimple } from '../component/field/TextFieldSimple.tsx';
import { ModalBanSteam, ModalUnbanSteam } from '../component/modal';
import { commonTableSearchSchema, isPermanentBan, LazyResult, RowsPerPage } from '../util/table.ts';
import { renderDate } from '../util/text.tsx';

const banSteamSearchSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z
        .enum(['ban_id', 'source_id', 'target_id', 'deleted', 'reason', 'created_on', 'valid_until', 'appeal_state'])
        .optional(),
    source_id: z.string().optional(),
    target_id: z.string().optional(),
    reason: z.nativeEnum(BanReason).optional(),
    appeal_state: z.nativeEnum(AppealState).optional(),
    deleted: z.boolean().optional()
});

export const Route = createFileRoute('/_mod/admin/ban/steam')({
    component: AdminBanSteam,
    validateSearch: (search) => banSteamSearchSchema.parse(search)
});

function AdminBanSteam() {
    const defaultRows = RowsPerPage.TwentyFive;
    const queryClient = useQueryClient();
    const { hasPermission } = Route.useRouteContext();
    const navigate = useNavigate({ from: Route.fullPath });
    const { page, rows, sortOrder, sortColumn, target_id, source_id, appeal_state, deleted } = Route.useSearch();
    const { data: bans, isLoading } = useQuery({
        queryKey: ['steamBans', { page, rows, sortOrder, sortColumn, target_id, source_id, appeal_state }],
        queryFn: async () => {
            return await apiGetBansSteam({
                limit: rows ?? defaultRows,
                offset: (page ?? 0) * (rows ?? defaultRows),
                order_by: sortColumn ?? 'ban_id',
                desc: (sortOrder ?? 'desc') == 'desc',
                source_id: source_id,
                target_id: target_id,
                appeal_state: appeal_state
            });
        }
    });

    const onNewBanSteam = async () => {
        const ban = await NiceModal.show<SteamBanRecord>(ModalBanSteam, {});
        queryClient.setQueryData(
            ['steamBans', { page, rows, sortOrder, sortColumn, target_id, source_id, appeal_state }],
            { data: [...(bans?.data ?? []), ban], count: (bans?.count ?? 0) + 1 }
        );
    };

    const onUnbanSteam = async (ban: SteamBanRecord) => {
        await NiceModal.show(ModalUnbanSteam, {
            banId: ban.ban_id,
            personaName: ban.target_personaname
        });
    };

    const onEditSteam = async (ban: SteamBanRecord) => {
        await NiceModal.show(ModalBanSteam, {
            banId: ban.ban_id,
            personaName: ban.target_personaname,
            existing: ban
        });
    };

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            await navigate({ to: '/admin/ban/steam', search: (prev) => ({ ...prev, ...value }) });
        },
        defaultValues: {
            source_id: source_id ?? '',
            target_id: target_id ?? '',
            appeal_state: appeal_state ?? AppealState.Any,
            deleted: deleted ?? false
        }
    });

    const clear = async () => {
        await navigate({
            to: '/admin/ban/steam',
            search: (prev) => ({ ...prev, source_id: '', target_id: '', appeal_state: AppealState.Any, deleted: false })
        });
    };

    return (
        <Grid container spacing={2}>
            <Grid xs={12}>
                <ContainerWithHeader title={'Filters'} iconLeft={<FilterListIcon />} marginTop={2}>
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
                                    children={(props) => {
                                        return (
                                            <TextFieldSimple {...props} label={'Subject Steam ID'} fullwidth={true} />
                                        );
                                    }}
                                />
                            </Grid>

                            <Grid xs={6} md={3}>
                                <Field
                                    name={'appeal_state'}
                                    children={(props) => {
                                        return (
                                            <SelectFieldSimple
                                                {...props}
                                                label={'Appeal State'}
                                                items={AppealStateCollection.map((i) => i)}
                                                renderMenu={(i) => {
                                                    return (
                                                        <MenuItem
                                                            value={i}
                                                            key={`${i}-${appealStateString(Number(i))}`}
                                                        >
                                                            {appealStateString(Number(i))}
                                                        </MenuItem>
                                                    );
                                                }}
                                            />
                                        );
                                    }}
                                />
                            </Grid>

                            <Grid xs="auto">
                                <Field
                                    name={'deleted'}
                                    children={(props) => {
                                        return (
                                            <CheckboxSimple
                                                {...props}
                                                label={'Incl. Deleted'}
                                                disabled={!hasPermission(PermissionLevel.User)}
                                            />
                                        );
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
                    title={'Steam Ban History'}
                    marginTop={0}
                    iconLeft={<GavelIcon />}
                    buttons={[
                        <Button
                            key={`ban-steam`}
                            variant={'contained'}
                            color={'success'}
                            startIcon={<AddIcon />}
                            sx={{ marginRight: 2 }}
                            onClick={onNewBanSteam}
                        >
                            Create
                        </Button>
                    ]}
                >
                    <Grid container spacing={3}>
                        <Grid xs={12}>
                            <BanSteamTable
                                bans={bans ?? { data: [], count: 0 }}
                                isLoading={isLoading}
                                onEditSteam={onEditSteam}
                                onUnbanSteam={onUnbanSteam}
                            />
                            <Paginator
                                page={page ?? 0}
                                rows={rows ?? defaultRows}
                                data={bans}
                                path={'/admin/ban/steam'}
                            />
                        </Grid>
                    </Grid>
                    {/*</Formik>*/}
                </ContainerWithHeaderAndButtons>
            </Grid>
        </Grid>
    );
}

const BanSteamTable = ({
    bans,
    isLoading,
    onEditSteam,
    onUnbanSteam
}: {
    bans: LazyResult<SteamBanRecord>;
    isLoading: boolean;
    onUnbanSteam: (ban: SteamBanRecord) => Promise<void>;
    onEditSteam: (ban: SteamBanRecord) => Promise<void>;
}) => {
    const columns = useMemo<ColumnDef<SteamBanRecord>[]>(
        () => [
            {
                accessorKey: 'ban_id',
                header: () => <TableHeadingCell name={'Ban ID'} />,
                cell: (info) => (
                    <Link component={RouterLink} to={`/ban/$ban_id`} params={{ ban_id: info.getValue() }}>
                        {`#${info.getValue()}`}
                    </Link>
                )
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
                            showCopy={true}
                            steam_id={bans.data[info.row.index].target_id}
                            personaname={bans.data[info.row.index].target_personaname}
                            avatar_hash={bans.data[info.row.index].target_avatarhash}
                        />
                    );
                }
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
                accessorKey: 'include_friends',
                header: () => <TableHeadingCell name={'F'} />,
                cell: (info) => <TableCellBool enabled={info.getValue() as boolean} />
            },
            {
                accessorKey: 'evade_ok',
                header: () => <TableHeadingCell name={'E'} />,
                cell: (info) => <TableCellBool enabled={info.getValue() as boolean} />
            },
            {
                accessorKey: 'report_id',
                header: () => <TableHeadingCell name={'Rep.'} />,
                cell: (info) =>
                    Boolean(info.getValue()) && (
                        <Link component={RouterLink} to={`/report/$reportId`} params={{ reportId: info.getValue() }}>
                            {`#${info.getValue()}`}
                        </Link>
                    )
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
                                    await onEditSteam(info.row.original);
                                }}
                            >
                                <Tooltip title={'Edit Ban'}>
                                    <EditIcon />
                                </Tooltip>
                            </IconButton>
                            <IconButton
                                color={'success'}
                                onClick={async () => {
                                    await onUnbanSteam(info.row.original);
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
        [bans.data, onEditSteam, onUnbanSteam]
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
