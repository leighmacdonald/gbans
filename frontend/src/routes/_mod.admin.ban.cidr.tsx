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
import { z } from 'zod';
import { apiGetBansCIDR, BanReason, BanReasons, CIDRBanRecord } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { DataTable } from '../component/DataTable.tsx';
import { Paginator } from '../component/Paginator.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { TableCellRelativeDateField } from '../component/TableCellRelativeDateField.tsx';
import { TableCellString } from '../component/TableCellString.tsx';
import { TableHeadingCell } from '../component/TableHeadingCell.tsx';
import { Title } from '../component/Title';
import { Buttons } from '../component/field/Buttons.tsx';
import { CheckboxSimple } from '../component/field/CheckboxSimple.tsx';
import { TextFieldSimple } from '../component/field/TextFieldSimple.tsx';
import { ModalBanCIDR, ModalUnbanCIDR } from '../component/modal';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { commonTableSearchSchema, isPermanentBan, LazyResult, RowsPerPage } from '../util/table.ts';
import { renderDate } from '../util/text.tsx';

const banCIDRSearchSchema = z.object({
    ...commonTableSearchSchema,
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
    const defaultRows = RowsPerPage.TwentyFive;
    const navigate = useNavigate({ from: Route.fullPath });
    const { page, rows, sortOrder, sortColumn, deleted, cidr, target_id, source_id } = Route.useSearch();
    const { data: bans, isLoading } = useQuery({
        queryKey: ['steamBans', { page, rows, sortOrder, sortColumn, target_id, source_id, cidr, deleted }],
        queryFn: async () => {
            return await apiGetBansCIDR({
                limit: rows ?? defaultRows,
                offset: (page ?? 0) * (rows ?? defaultRows),
                order_by: sortColumn,
                desc: (sortOrder ?? 'desc') == 'desc',
                source_id: source_id,
                target_id: target_id,
                deleted: deleted,
                ip: cidr
            });
        }
    });
    // const [newCIDRBans, setNewCIDRBans] = useState<CIDRBanRecord[]>([]);
    // const { sendFlash } = useUserFlashCtx();

    const onNewBanCIDR = async () => {
        try {
            const ban = await NiceModal.show<CIDRBanRecord>(ModalBanCIDR, {});
            sendFlash('success', `Created CIDR ban successfully #${ban.net_id}`);
        } catch (e) {
            sendFlash('error', `${e}`);
        }
    };

    const onUnbanCIDR = async (ban: CIDRBanRecord) => {
        await NiceModal.show(ModalUnbanCIDR, {
            banId: ban.net_id,
            personaName: ban.target_personaname
        });
    };

    const onEditCIDR = async (ban: CIDRBanRecord) => {
        await NiceModal.show(ModalBanCIDR, {
            banId: ban.net_id,
            personaName: ban.target_personaname,
            existing: ban
        });
    };

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            await navigate({ to: '/admin/ban/cidr', search: (prev) => ({ ...prev, ...value }) });
        },
        defaultValues: {
            source_id: source_id ?? '',
            target_id: target_id ?? '',
            cidr: cidr ?? '',
            deleted: deleted ?? false
        }
    });
    const clear = async () => {
        await navigate({
            to: '/admin/ban/cidr',
            search: (prev) => ({ ...prev, source_id: '', target_id: '', cidr: '', deleted: false })
        });
    };

    return (
        <Grid container>
            <Title>Ban CIDR</Title>
            <Grid xs={12} marginBottom={2}>
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
                                        return (
                                            <TextFieldSimple {...props} label={'Author Steam ID'} fullwidth={true} />
                                        );
                                    }}
                                />
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
                                    name={'cidr'}
                                    children={(props) => {
                                        return <TextFieldSimple {...props} label={'IP/CIDR'} fullwidth={true} />;
                                    }}
                                />
                            </Grid>

                            <Grid xs={6} md={3}>
                                <Field
                                    name={'deleted'}
                                    children={(props) => {
                                        return <CheckboxSimple {...props} label={'Incl. Deleted'} />;
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
                    <BanCIDRTable
                        bans={bans ?? { data: [], count: 0 }}
                        isLoading={isLoading}
                        onEditCIDR={onEditCIDR}
                        onUnbanCIDR={onUnbanCIDR}
                    />
                    <Paginator data={bans} page={page ?? 0} rows={rows ?? defaultRows} path={Route.fullPath} />
                </ContainerWithHeaderAndButtons>
            </Grid>
        </Grid>
    );
}

const BanCIDRTable = ({
    bans,
    isLoading,
    onEditCIDR,
    onUnbanCIDR
}: {
    bans: LazyResult<CIDRBanRecord>;
    isLoading: boolean;
    onUnbanCIDR: (ban: CIDRBanRecord) => Promise<void>;
    onEditCIDR: (ban: CIDRBanRecord) => Promise<void>;
}) => {
    const columns = useMemo<ColumnDef<CIDRBanRecord>[]>(
        () => [
            {
                accessorKey: 'net_id',
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
                            showCopy={true}
                            steam_id={bans.data[info.row.index].target_id}
                            personaname={bans.data[info.row.index].target_personaname}
                            avatar_hash={bans.data[info.row.index].target_avatarhash}
                        />
                    );
                }
            },
            {
                accessorKey: 'cidr',
                header: () => <TableHeadingCell name={'CIDR (hosts)'} />,
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
                                    await onEditCIDR(info.row.original);
                                }}
                            >
                                <Tooltip title={'Edit Ban'}>
                                    <EditIcon />
                                </Tooltip>
                            </IconButton>
                            <IconButton
                                color={'success'}
                                onClick={async () => {
                                    await onUnbanCIDR(info.row.original);
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
        [bans.data, onEditCIDR, onUnbanCIDR]
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
