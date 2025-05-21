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
import { apiGetBansCIDR } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { Title } from '../component/Title';
import { ModalBanCIDR, ModalUnbanCIDR } from '../component/modal';
import { FullTable } from '../component/table/FullTable.tsx';
import { TableCellBool } from '../component/table/TableCellBool.tsx';
import { TableCellRelativeDateField } from '../component/table/TableCellRelativeDateField.tsx';
import { TableCellString } from '../component/table/TableCellString.tsx';
import { useAppForm } from '../contexts/formContext.tsx';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { AppealState, BanReasonEnum, BanReasons, CIDRBanRecord } from '../schema/bans.ts';
import { commonTableSearchSchema, initColumnFilter, initPagination, isPermanentBan } from '../util/table.ts';
import { renderDate } from '../util/time.ts';
import { makeValidateSteamIDCallback } from '../util/validator/makeValidateSteamIDCallback.ts';

const banCIDRSearchSchema = commonTableSearchSchema.extend({
    sortColumn: z
        .enum(['net_id', 'source_id', 'target_id', 'deleted', 'reason', 'created_on', 'valid_until'])
        .optional(),
    source_id: z.string().optional(),
    target_id: z.string().optional(),
    cidr: z.string().optional(),
    deleted: z.boolean().optional()
});

export const Route = createFileRoute('/_mod/admin/ban/cidr')({
    component: AdminBanCIDR,
    validateSearch: (search) => banCIDRSearchSchema.parse(search)
});

function AdminBanCIDR() {
    const { sendFlash } = useUserFlashCtx();
    const navigate = useNavigate({ from: Route.fullPath });
    const search = Route.useSearch();
    const [pagination, setPagination] = useState<PaginationState>(initPagination(search.pageIndex, search.pageSize));
    const [sorting] = useState<SortingState>([{ id: 'net_id', desc: true }]);
    const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>(initColumnFilter(search));

    const { data: bans, isLoading } = useQuery({
        queryKey: ['cidrBans'],
        queryFn: async () => {
            return await apiGetBansCIDR({ deleted: search.deleted ?? false, appeal_state: AppealState.Any });
        }
    });

    const onNewBanCIDR = async () => {
        try {
            const ban = await NiceModal.show<CIDRBanRecord>(ModalBanCIDR, {});
            sendFlash('success', `Created CIDR ban successfully #${ban.net_id}`);
        } catch (e) {
            sendFlash('error', `${e}`);
        }
    };

    const form = useAppForm({
        onSubmit: async ({ value }) => {
            setColumnFilters(initColumnFilter(value));
            await navigate({ to: '/admin/ban/cidr', search: (prev) => ({ ...prev, ...value }) });
        },
        validators: {
            onChangeAsyncDebounceMs: 500,
            onChangeAsync: z.object({
                source_id: makeValidateSteamIDCallback(),
                target_id: makeValidateSteamIDCallback(),
                cidr: z.string(),
                deleted: z.boolean()
            })
        },
        defaultValues: {
            source_id: search.source_id ?? '',
            target_id: search.target_id ?? '',
            cidr: search.cidr ?? '',
            deleted: search.deleted ?? false
        }
    });

    const clear = async () => {
        form.reset();
        setColumnFilters([]);
        await navigate({
            to: '/admin/ban/cidr',
            search: (prev) => ({ ...prev, source_id: '', target_id: '', cidr: '', deleted: false })
        });
    };

    const columns = useMemo(() => {
        const onUnban = async (ban: CIDRBanRecord) => {
            await NiceModal.show(ModalUnbanCIDR, {
                banId: ban.net_id,
                personaName: ban.target_personaname
            });
        };

        const onEdit = async (ban: CIDRBanRecord) => {
            await NiceModal.show(ModalBanCIDR, {
                banId: ban.net_id,
                personaName: ban.target_personaname,
                existing: ban
            });
        };

        return makeColumns(onEdit, onUnban);
    }, []);

    return (
        <Grid container>
            <Title>Ban CIDR</Title>
            <Grid size={{ xs: 12 }} marginBottom={2}>
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
                                    name={'cidr'}
                                    children={(field) => {
                                        return <field.TextField label={'IP/CIDR'} />;
                                    }}
                                />
                            </Grid>

                            <Grid size={{ xs: 6, md: 3 }}>
                                <form.AppField
                                    name={'deleted'}
                                    children={(field) => {
                                        return <field.CheckboxField label={'Incl. Deleted'} />;
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
                    title={'CIDR Ban History'}
                    marginTop={0}
                    iconLeft={<GavelIcon />}
                    buttons={[
                        <Button
                            key={'btn-cidr'}
                            variant={'contained'}
                            color={'success'}
                            startIcon={<AddIcon />}
                            sx={{ marginRight: 2 }}
                            onClick={onNewBanCIDR}
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

const columnHelper = createColumnHelper<CIDRBanRecord>();

const makeColumns = (onEdit: (ban: CIDRBanRecord) => Promise<void>, onUnban: (ban: CIDRBanRecord) => Promise<void>) => [
    columnHelper.accessor('net_id', {
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
    columnHelper.accessor('cidr', {
        header: 'CIDR (hosts)',
        size: 150,
        cell: (info) => <Typography>{`${info.getValue()}`}</Typography>
    }),
    columnHelper.accessor('reason', {
        header: 'Reason',
        size: 150,
        cell: (info) => <Typography>{BanReasons[info.getValue() as BanReasonEnum]}</Typography>
    }),
    columnHelper.accessor('created_on', {
        header: 'Created',
        size: 100,
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
    columnHelper.accessor('deleted', {
        header: 'D',
        size: 32,
        cell: (info) => <TableCellBool enabled={info.getValue()} />
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
