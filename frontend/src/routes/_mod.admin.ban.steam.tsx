import { useMemo, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import AddIcon from '@mui/icons-material/Add';
import EditIcon from '@mui/icons-material/Edit';
import FilterListIcon from '@mui/icons-material/FilterList';
import GavelIcon from '@mui/icons-material/Gavel';
import UndoIcon from '@mui/icons-material/Undo';
import Button from '@mui/material/Button';
import IconButton from '@mui/material/IconButton';
import Link from '@mui/material/Link';
import MenuItem from '@mui/material/MenuItem';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { ColumnFiltersState, createColumnHelper, PaginationState, SortingState } from '@tanstack/react-table';
import { zodValidator } from '@tanstack/zod-form-adapter';
import { z } from 'zod';
import { apiGetBansSteam, BanReason, BanReasons, banReasonsCollection, SteamBanRecord } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { FullTable } from '../component/FullTable.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import RouterLink from '../component/RouterLink.tsx';
import { TableCellBool } from '../component/TableCellBool.tsx';
import { TableCellRelativeDateField } from '../component/TableCellRelativeDateField.tsx';
import { TableHeadingCell } from '../component/TableHeadingCell.tsx';
import { Title } from '../component/Title';
import { Buttons } from '../component/field/Buttons.tsx';
import { CheckboxSimple } from '../component/field/CheckboxSimple.tsx';
import { SelectFieldSimple } from '../component/field/SelectFieldSimple.tsx';
import { TextFieldSimple } from '../component/field/TextFieldSimple.tsx';
import { ModalBanSteam, ModalUnbanSteam } from '../component/modal';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { initColumnFilter, initPagination, isPermanentBan, makeCommonTableSearchSchema } from '../util/table.ts';
import { renderDate } from '../util/text.tsx';

const banSteamSearchSchema = z.object({
    ...makeCommonTableSearchSchema([
        'ban_id',
        'source_id',
        'target_id',
        'as_num',
        'reason',
        'created_on',
        'updated_on'
    ]),
    source_id: z.string().optional(),
    target_id: z.string().optional(),
    reason: z.nativeEnum(BanReason).optional(),
    deleted: z.boolean().optional()
});

export const Route = createFileRoute('/_mod/admin/ban/steam')({
    component: AdminBanSteam,
    validateSearch: (search) => banSteamSearchSchema.parse(search)
});

const queryKey = ['steamBans'];

function AdminBanSteam() {
    const queryClient = useQueryClient();
    const navigate = useNavigate({ from: Route.fullPath });
    const search = Route.useSearch();
    const [pagination, setPagination] = useState<PaginationState>(initPagination(search.pageIndex, search.pageSize));
    const [sorting] = useState<SortingState>([{ id: 'ban_id', desc: true }]);
    const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>(initColumnFilter(search));
    const { sendFlash } = useUserFlashCtx();

    const { data: bans, isLoading } = useQuery({
        queryKey: queryKey,
        queryFn: async () => {
            return await apiGetBansSteam({
                deleted: true
            });
        }
    });

    const onNewBanSteam = async () => {
        try {
            const ban = await NiceModal.show<SteamBanRecord>(ModalBanSteam, {});
            queryClient.setQueryData(queryKey, [...(bans ?? []), ban]);
        } catch (e) {
            sendFlash('error', `Error trying to setup ban: ${e}`);
        }
    };

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            setColumnFilters(initColumnFilter(value));
            await navigate({
                to: '/admin/ban/steam',
                search: (prev) => ({ ...prev, ...value })
            });
        },
        validatorAdapter: zodValidator,
        defaultValues: {
            source_id: search.source_id ?? '',
            target_id: search.target_id ?? '',
            reason: search.reason ?? BanReason.Any,
            deleted: search.deleted ?? false
        }
    });

    const clear = async () => {
        setColumnFilters([]);
        reset();
        await navigate({
            to: '/admin/ban/steam',
            search: (prev) => ({
                ...prev,
                source_id: undefined,
                target_id: undefined,
                reason: undefined,
                valid_until: undefined
            })
        });
    };

    const columns = useMemo(() => {
        const onUnban = async (ban: SteamBanRecord) => {
            try {
                await NiceModal.show(ModalUnbanSteam, {
                    banId: ban.ban_id,
                    personaName: ban.target_personaname
                });
                queryClient.setQueryData(
                    queryKey,
                    (bans ?? []).filter((b) => b.ban_id != ban.ban_id)
                );
                sendFlash('success', 'Unbanned player successfully');
            } catch (e) {
                sendFlash('error', `Error trying to unban: ${e}`);
            }
        };

        const onEdit = async (ban: SteamBanRecord) => {
            try {
                const updated = await NiceModal.show<SteamBanRecord>(ModalBanSteam, {
                    banId: ban.ban_id,
                    personaName: ban.target_personaname,
                    existing: ban
                });
                queryClient.setQueryData(
                    queryKey,
                    (bans ?? []).map((b) => (b.ban_id == updated.ban_id ? updated : b))
                );
            } catch (e) {
                sendFlash('error', `Error trying to edit ban: ${e}`);
            }
        };

        return makeColumns(onEdit, onUnban);
    }, [bans, queryClient, sendFlash]);

    return (
        <Grid container spacing={2}>
            <Title>Ban SteamID</Title>
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
                            <Grid xs={6} md={3}>
                                <Field
                                    name={'source_id'}
                                    children={(props) => {
                                        return <TextFieldSimple {...props} label={'Author Steam ID'} />;
                                    }}
                                />
                            </Grid>

                            <Grid xs={6} md={3}>
                                <Field
                                    name={'target_id'}
                                    children={(props) => {
                                        return <TextFieldSimple {...props} label={'Subject Steam ID'} />;
                                    }}
                                />
                            </Grid>

                            <Grid xs={6} md={3}>
                                <Field
                                    name={'reason'}
                                    children={(props) => {
                                        return (
                                            <SelectFieldSimple
                                                {...props}
                                                label={'Ban Reason'}
                                                items={banReasonsCollection}
                                                renderMenu={(i) => {
                                                    if (i == undefined) {
                                                        return null;
                                                    }
                                                    return (
                                                        <MenuItem value={i} key={`${i}-${BanReasons[i]}`}>
                                                            {BanReasons[i]}
                                                        </MenuItem>
                                                    );
                                                }}
                                            />
                                        );
                                    }}
                                />
                            </Grid>
                            <Grid xs={6} md={3}>
                                <Field
                                    name={'deleted'}
                                    children={(props) => {
                                        return <CheckboxSimple {...props} label={'Show deleted/expired'} />;
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
                            <FullTable
                                columnFilters={columnFilters}
                                pagination={pagination}
                                setPagination={setPagination}
                                data={bans ?? []}
                                isLoading={isLoading}
                                columns={columns}
                                sorting={sorting}
                            />
                        </Grid>
                    </Grid>
                </ContainerWithHeaderAndButtons>
            </Grid>
        </Grid>
    );
}

const columnHelper = createColumnHelper<SteamBanRecord>();

const makeColumns = (
    onEdit: (ban: SteamBanRecord) => Promise<void>,
    onUnban: (ban: SteamBanRecord) => Promise<void>
) => [
    columnHelper.accessor('ban_id', {
        enableColumnFilter: false,
        header: () => <TableHeadingCell name={'Ban ID'} />,
        cell: (info) => (
            <Link component={RouterLink} to={`/ban/$ban_id`} params={{ ban_id: info.getValue() }}>
                {`#${info.getValue()}`}
            </Link>
        )
    }),
    columnHelper.accessor('source_id', {
        header: () => <TableHeadingCell name={'Author'} />,
        cell: (info) => {
            return typeof info.row.original === 'undefined' ? (
                ''
            ) : (
                <PersonCell
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
                    showCopy={true}
                    steam_id={info.row.original.target_id}
                    personaname={info.row.original.target_personaname}
                    avatar_hash={info.row.original.target_avatarhash}
                />
            );
        }
    }),
    columnHelper.accessor('reason', {
        enableColumnFilter: true,
        filterFn: (row, _, filterValue) => {
            return filterValue == BanReason.Any || row.original.reason == filterValue;
        },
        header: () => <TableHeadingCell name={'Reason'} />,
        cell: (info) => <Typography>{BanReasons[info.getValue() as BanReason]}</Typography>
    }),
    columnHelper.accessor('created_on', {
        header: () => <TableHeadingCell name={'Created'} />,
        cell: (info) => <Typography>{renderDate(info.getValue() as Date)}</Typography>
    }),
    columnHelper.accessor('valid_until', {
        header: () => <TableHeadingCell name={'Duration'} />,
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
    columnHelper.accessor('include_friends', {
        header: () => <TableHeadingCell name={'F'} tooltip={'Friends list also banned'} />,
        cell: (info) => <TableCellBool enabled={info.getValue()} />
    }),
    columnHelper.accessor('evade_ok', {
        header: () => (
            <TableHeadingCell
                name={'E'}
                tooltip={'Evasion OK. Players connecting from the same ip will not be banned.'}
            />
        ),
        cell: (info) => <TableCellBool enabled={info.getValue()} />
    }),
    columnHelper.accessor('deleted', {
        filterFn: (row, _, filterValue) => {
            return filterValue ? true : !row.original.deleted;
        },
        header: () => <TableHeadingCell name={'D'} tooltip={'Deleted / Expired Bans'} />,
        cell: (info) => <TableCellBool enabled={info.getValue()} />
    }),
    columnHelper.accessor('report_id', {
        header: () => <TableHeadingCell name={'Rep.'} />,
        cell: (info) =>
            Boolean(info.getValue()) && (
                <Link component={RouterLink} to={`/report/$reportId`} params={{ reportId: info.getValue() }}>
                    {`#${info.getValue()}`}
                </Link>
            )
    }),
    columnHelper.display({
        id: 'edit',
        cell: (info) => (
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
        )
    }),
    columnHelper.display({
        id: 'unban',
        cell: (info) => (
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
        )
    })
];
