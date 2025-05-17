import { useMemo, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import AddIcon from '@mui/icons-material/Add';
import EditIcon from '@mui/icons-material/Edit';
import FilterListIcon from '@mui/icons-material/FilterList';
import GavelIcon from '@mui/icons-material/Gavel';
import UndoIcon from '@mui/icons-material/Undo';
import Button from '@mui/material/Button';
import Grid from '@mui/material/Grid';
import IconButton from '@mui/material/IconButton';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { ColumnFiltersState, createColumnHelper, PaginationState, SortingState } from '@tanstack/react-table';
import { z } from 'zod';
import { apiGetBansASN, ASNBanRecord, BanReason, BanReasons } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { Title } from '../component/Title.tsx';
import { ModalBanASN, ModalUnbanASN } from '../component/modal';
import { FullTable } from '../component/table/FullTable.tsx';
import { TableCellRelativeDateField } from '../component/table/TableCellRelativeDateField.tsx';
import { TableCellString } from '../component/table/TableCellString.tsx';
import { useAppForm } from '../contexts/formContext.tsx';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { initColumnFilter, initPagination, isPermanentBan, makeCommonTableSearchSchema } from '../util/table.ts';
import { renderDate } from '../util/time.ts';
import { makeValidateSteamIDCallback } from '../util/validator/makeValidateSteamIDCallback.ts';

const banASNSearchSchema = z.object({
    ...makeCommonTableSearchSchema([
        'ban_asn_id',
        'source_id',
        'target_id',
        'deleted',
        'reason',
        'as_num',
        'valid_until'
    ]),
    source_id: z.string().optional(),
    target_id: z.string().optional(),
    as_num: z.string().optional(),
    deleted: z.boolean().optional()
});

export const Route = createFileRoute('/_mod/admin/ban/asn')({
    component: AdminBanASN,
    validateSearch: (search) => banASNSearchSchema.parse(search)
});

const queryKey = ['asnBans'];

function AdminBanASN() {
    const queryClient = useQueryClient();
    const { sendFlash } = useUserFlashCtx();
    const navigate = useNavigate({ from: Route.fullPath });
    const search = Route.useSearch();
    const [pagination, setPagination] = useState<PaginationState>(initPagination(search.pageIndex, search.pageSize));
    const [sorting] = useState<SortingState>([{ id: 'ban_asn_id', desc: true }]);
    const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>(initColumnFilter(search));

    const { data: bans, isLoading } = useQuery({
        queryKey: queryKey,
        queryFn: async () => {
            return await apiGetBansASN({ deleted: search.deleted ?? false });
        }
    });

    const form = useAppForm({
        onSubmit: async ({ value }) => {
            setColumnFilters(initColumnFilter(value));
            await navigate({
                to: '/admin/ban/asn',
                search: (prev) => ({ ...prev, ...value })
            });
        },
        validators: {
            onChangeAsyncDebounceMs: 500,
            onChangeAsync: z.object({
                source_id: makeValidateSteamIDCallback(),
                target_id: makeValidateSteamIDCallback(),
                as_num: z.string(),
                deleted: z.boolean()
            })
        },
        defaultValues: {
            source_id: search.source_id ?? '',
            target_id: search.target_id ?? '',
            as_num: search.as_num ?? '',
            deleted: search.deleted ?? false
        }
    });

    const clear = async () => {
        form.reset();
        setColumnFilters([]);
        await navigate({
            to: '/admin/ban/asn',
            search: (prev) => ({
                ...prev,
                source_id: undefined,
                target_id: undefined,
                as_num: undefined,
                deleted: undefined
            })
        });
    };

    const onNewBanASN = async () => {
        try {
            const ban = await NiceModal.show<ASNBanRecord>(ModalBanASN, {});
            sendFlash('success', `Created ASN ban successfully #${ban.ban_asn_id}`);
        } catch (e) {
            sendFlash('error', `${e}`);
        }
    };

    const columns = useMemo(() => {
        const onUnban = async (ban: ASNBanRecord) => {
            try {
                await NiceModal.show(ModalUnbanASN, {
                    banId: ban.ban_asn_id
                });
                queryClient.setQueryData(
                    queryKey,
                    (bans ?? []).filter((b) => b.ban_asn_id != ban.ban_asn_id)
                );
                sendFlash('success', 'Unbanned player successfully');
            } catch (e) {
                sendFlash('error', `Error trying to unban: ${e}`);
            }
        };

        const onEdit = async (ban: ASNBanRecord) => {
            try {
                const updated = await NiceModal.show<ASNBanRecord>(ModalBanASN, {
                    banId: ban.ban_asn_id,
                    existing: ban
                });
                queryClient.setQueryData(
                    queryKey,
                    (bans ?? []).map((b) => (b.ban_asn_id == updated.ban_asn_id ? updated : b))
                );
            } catch (e) {
                sendFlash('error', `Error trying to edit ban: ${e}`);
            }
        };
        return makeColumns(onEdit, onUnban);
    }, [bans, queryClient, sendFlash]);

    return (
        <Grid container spacing={2}>
            <Title>Ban ASN</Title>
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
                            <Grid size={{ xs: 4 }}>
                                <Grid size={{ xs: 6, md: 3 }}>
                                    <form.AppField
                                        name={'source_id'}
                                        children={(field) => {
                                            return <field.TextField label={'Author Steam ID'} />;
                                        }}
                                    />
                                </Grid>
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
                                    name={'as_num'}
                                    children={(field) => {
                                        return <field.TextField label={'AS Number'} />;
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
                                <form.ClearButton onClick={clear} />
                                <form.ResetButton />
                                <form.SubmitButton />
                            </Grid>
                        </Grid>
                    </form>
                </ContainerWithHeader>
            </Grid>

            <Grid size={{ xs: 12 }}>
                <ContainerWithHeaderAndButtons
                    title={'ASN Ban History'}
                    marginTop={0}
                    iconLeft={<GavelIcon />}
                    buttons={[
                        <Button
                            key={'btn-asn'}
                            variant={'contained'}
                            color={'success'}
                            startIcon={<AddIcon />}
                            sx={{ marginRight: 2 }}
                            onClick={onNewBanASN}
                        >
                            Create
                        </Button>
                    ]}
                >
                    <FullTable
                        data={bans ?? []}
                        isLoading={isLoading}
                        columns={columns}
                        sorting={sorting}
                        pagination={pagination}
                        setPagination={setPagination}
                        columnFilters={columnFilters}
                        toOptions={{ from: Route.fullPath }}
                    />
                </ContainerWithHeaderAndButtons>
            </Grid>
        </Grid>
    );
}

const columnHelper = createColumnHelper<ASNBanRecord>();

const makeColumns = (onEdit: (ban: ASNBanRecord) => Promise<void>, onUnban: (ban: ASNBanRecord) => Promise<void>) => [
    columnHelper.accessor('ban_asn_id', {
        header: () => 'Ban ID',
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
                    showCopy={true}
                    steam_id={info.row.original.target_id}
                    personaname={info.row.original.target_personaname}
                    avatar_hash={info.row.original.target_avatarhash}
                />
            );
        }
    }),
    columnHelper.accessor('as_num', {
        size: 100,
        header: 'AS Number',
        cell: (info) => <Typography>{`${info.getValue()}`}</Typography>
    }),
    columnHelper.accessor('reason', {
        size: 100,
        header: () => 'Reason',
        cell: (info) => <Typography>{BanReasons[info.getValue() as BanReason]}</Typography>
    }),
    columnHelper.accessor('created_on', {
        size: 100,
        header: () => 'Created',
        cell: (info) => <Typography>{renderDate(info.getValue() as Date)}</Typography>
    }),
    columnHelper.accessor('valid_until', {
        header: 'Expires',
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
