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
import { apiGetBansASN, ASNBanRecord, BanReason, BanReasons } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { FullTable } from '../component/FullTable.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { TableCellRelativeDateField } from '../component/TableCellRelativeDateField.tsx';
import { TableCellString } from '../component/TableCellString.tsx';
import { TableHeadingCell } from '../component/TableHeadingCell.tsx';
import { Title } from '../component/Title.tsx';
import { Buttons } from '../component/field/Buttons.tsx';
import { CheckboxSimple } from '../component/field/CheckboxSimple.tsx';
import { TextFieldSimple } from '../component/field/TextFieldSimple.tsx';
import { ModalBanASN, ModalUnbanASN } from '../component/modal';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { initColumnFilter, initPagination, isPermanentBan, makeCommonTableSearchSchema } from '../util/table.ts';
import { renderDate } from '../util/text.tsx';
import { makeSteamidValidatorsOptional } from '../util/validator/makeSteamidValidatorsOptional.ts';

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

function AdminBanASN() {
    const { sendFlash } = useUserFlashCtx();
    const navigate = useNavigate({ from: Route.fullPath });
    const search = Route.useSearch();
    const [pagination, setPagination] = useState<PaginationState>(initPagination(search.pageIndex, search.pageSize));
    const [sorting] = useState<SortingState>([{ id: 'ban_asn_id', desc: true }]);
    const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>(initColumnFilter(search));

    const { data: bans, isLoading } = useQuery({
        queryKey: ['asnBans', { deleted: search.deleted }],
        queryFn: async () => {
            return await apiGetBansASN({ deleted: search.deleted ?? false });
        }
    });

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            setColumnFilters(initColumnFilter(value));
            await navigate({
                to: '/admin/ban/asn',
                search: (prev) => ({ ...prev, ...value })
            });
        },
        validatorAdapter: zodValidator,
        validators: {
            onChange: banASNSearchSchema
        },
        defaultValues: {
            source_id: search.source_id ?? '',
            target_id: search.target_id ?? '',
            as_num: search.as_num ?? '',
            deleted: search.deleted ?? false
        }
    });

    const clear = async () => {
        reset();
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
            await NiceModal.show(ModalUnbanASN, {
                banId: ban.ban_asn_id
            });
        };

        const onEdit = async (ban: ASNBanRecord) => {
            await NiceModal.show(ModalBanASN, {
                banId: ban.ban_asn_id,
                existing: ban
            });
        };
        return makeColumns(onEdit, onUnban);
    }, []);

    return (
        <Grid container spacing={2}>
            <Title>Ban ASN</Title>
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
                                    name={'as_num'}
                                    children={(props) => {
                                        return <TextFieldSimple {...props} label={'AS Number'} fullwidth={true} />;
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
                    />
                </ContainerWithHeaderAndButtons>
            </Grid>
        </Grid>
    );
}

const columnHelper = createColumnHelper<ASNBanRecord>();

const makeColumns = (onEdit: (ban: ASNBanRecord) => Promise<void>, onUnban: (ban: ASNBanRecord) => Promise<void>) => [
    columnHelper.accessor('ban_asn_id', {
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
                    showCopy={true}
                    steam_id={info.row.original.target_id}
                    personaname={info.row.original.target_personaname}
                    avatar_hash={info.row.original.target_avatarhash}
                />
            );
        }
    }),
    columnHelper.accessor('as_num', {
        header: () => <TableHeadingCell name={'AS Number'} />,
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
