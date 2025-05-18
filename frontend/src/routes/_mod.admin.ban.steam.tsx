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
import MenuItem from '@mui/material/MenuItem';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { ColumnFiltersState, createColumnHelper, PaginationState, SortingState } from '@tanstack/react-table';
import { z } from 'zod';
import { apiGetBansSteam } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { TextLink } from '../component/TextLink.tsx';
import { Title } from '../component/Title';
import { ModalBanSteam, ModalUnbanSteam } from '../component/modal';
import { FullTable } from '../component/table/FullTable.tsx';
import { TableCellBool } from '../component/table/TableCellBool.tsx';
import { TableCellRelativeDateField } from '../component/table/TableCellRelativeDateField.tsx';
import { useAppForm } from '../contexts/formContext.tsx';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import {
    AppealState,
    BanReason,
    BanReasonEnum,
    BanReasons,
    banReasonsCollection,
    SteamBanRecord
} from '../schema/bans.ts';
import { initColumnFilter, initPagination, isPermanentBan, makeCommonTableSearchSchema } from '../util/table.ts';
import { renderDate } from '../util/time.ts';

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
                deleted: true,
                appeal_state: AppealState.Any
            });
        }
    });

    const onNewBanSteam = async () => {
        try {
            const ban = await NiceModal.show<SteamBanRecord>(ModalBanSteam, {});
            queryClient.setQueryData(queryKey, [...(bans ?? []), ban]);
        } catch (e) {
            sendFlash('error', `Error trying to set up ban: ${e}`);
        }
    };

    const form = useAppForm({
        onSubmit: async ({ value }) => {
            setColumnFilters(initColumnFilter(value));
            await navigate({
                to: '/admin/ban/steam',
                search: (prev) => ({ ...prev, ...value })
            });
        },
        defaultValues: {
            source_id: search.source_id ?? '',
            target_id: search.target_id ?? '',
            reason: search.reason ?? BanReason.Any,
            deleted: search.deleted ?? false
        }
    });

    const clear = async () => {
        setColumnFilters([]);
        form.reset();
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
            <Grid size={{ xs: 12 }}>
                <ContainerWithHeader title={'Filters'} iconLeft={<FilterListIcon />} marginTop={2}>
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
                                        return <field.TextField label={'Author Steam ID'} />;
                                    }}
                                />
                            </Grid>

                            <Grid size={{ xs: 6, md: 3 }}>
                                <form.AppField
                                    name={'target_id'}
                                    children={(field) => {
                                        return <field.TextField label={'Subject Steam ID'} />;
                                    }}
                                />
                            </Grid>

                            <Grid size={{ xs: 6, md: 3 }}>
                                <form.AppField
                                    name={'reason'}
                                    children={(field) => {
                                        return (
                                            <field.SelectField
                                                label={'Ban Reason'}
                                                items={banReasonsCollection}
                                                renderItem={(i) => {
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
                            <Grid size={{ xs: 6, md: 3 }}>
                                <form.AppField
                                    name={'deleted'}
                                    children={(field) => {
                                        return <field.CheckboxField label={'Show deleted/expired'} />;
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
                    <Grid container spacing={2}>
                        <Grid size={{ xs: 12 }}>
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
        size: 50,
        header: 'Ban ID',
        cell: (info) => (
            <TextLink to={`/ban/$ban_id`} params={{ ban_id: String(info.getValue()) }}>
                {`#${info.getValue()}`}
            </TextLink>
        )
    }),
    columnHelper.accessor('source_id', {
        header: 'Author',
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
        header: 'Subject',
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
        size: 150,
        filterFn: (row, _, filterValue) => {
            return filterValue == BanReason.Any || row.original.reason == filterValue;
        },
        header: 'Reason',
        cell: (info) => <Typography>{BanReasons[info.getValue() as BanReasonEnum]}</Typography>
    }),
    columnHelper.accessor('created_on', {
        header: 'Created',
        size: 100,
        cell: (info) => <Typography>{renderDate(info.getValue() as Date)}</Typography>
    }),
    columnHelper.accessor('valid_until', {
        header: 'Duration',
        size: 100,
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
        meta: { tooltip: 'Friends list also banned' },
        size: 30,
        header: 'F',
        cell: (info) => <TableCellBool enabled={info.getValue()} />
    }),
    columnHelper.accessor('evade_ok', {
        meta: { tooltip: 'Evasion OK. Players connecting from the same ip will not be banned.' },
        size: 30,
        header: 'E',
        cell: (info) => <TableCellBool enabled={info.getValue()} />
    }),
    columnHelper.accessor('deleted', {
        size: 30,
        meta: { tooltip: 'Deleted / Expired Bans' },
        filterFn: (row, _, filterValue) => {
            return filterValue ? true : !row.original.deleted;
        },
        header: 'D',
        cell: (info) => <TableCellBool enabled={info.getValue()} />
    }),
    columnHelper.accessor('report_id', {
        header: 'Rep.',
        size: 60,
        meta: { tooltip: 'Linked report' },
        cell: (info) =>
            Boolean(info.getValue()) && (
                <TextLink to={`/report/$reportId`} params={{ reportId: String(info.getValue()) }}>
                    {`#${info.getValue()}`}
                </TextLink>
            )
    }),
    columnHelper.display({
        id: 'edit',
        size: 30,
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
        size: 30,
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
