import { useState } from 'react';
import FilterListIcon from '@mui/icons-material/FilterList';
import MenuItem from '@mui/material/MenuItem';
import TableCell from '@mui/material/TableCell';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import {
    ColumnFiltersState,
    createColumnHelper,
    getCoreRowModel,
    getFilteredRowModel,
    getPaginationRowModel,
    getSortedRowModel,
    SortingState,
    useReactTable
} from '@tanstack/react-table';
import { zodValidator } from '@tanstack/zod-form-adapter';
import { z } from 'zod';
import {
    apiGetAppeals,
    AppealState,
    AppealStateCollection,
    appealStateString,
    BanReasons,
    SteamBanRecord
} from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { DataTable } from '../component/DataTable.tsx';
import { PaginatorLocal } from '../component/PaginatorLocal.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { TextLink } from '../component/TextLink.tsx';
import { Title } from '../component/Title';
import { Buttons } from '../component/field/Buttons.tsx';
import { SelectFieldSimple } from '../component/field/SelectFieldSimple.tsx';
import { TextFieldSimple } from '../component/field/TextFieldSimple.tsx';
import { TablePropsAll } from '../types/table.ts';
import { commonTableSearchSchema, initColumnFilter, initPagination, initSortOrder } from '../util/table.ts';
import { renderDateTime } from '../util/time.ts';
import { makeSteamidValidatorsOptional } from '../util/validator/makeSteamidValidatorsOptional.ts';

const appealSearchSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z
        .enum(['report_id', 'source_id', 'target_id', 'appeal_state', 'reason', 'created_on', 'updated_on'])
        .optional(),
    source_id: z.string().optional(),
    target_id: z.string().optional(),
    appeal_state: z.nativeEnum(AppealState).optional()
});

export const Route = createFileRoute('/_mod/admin/appeals')({
    component: AdminAppeals,
    validateSearch: (search) => appealSearchSchema.parse(search)
});

function AdminAppeals() {
    const navigate = useNavigate({ from: Route.fullPath });
    const search = Route.useSearch();
    const [pagination, setPagination] = useState(initPagination(search.pageIndex, search.pageSize));
    const [sorting, setSorting] = useState<SortingState>(
        initSortOrder(search.sortColumn, search.sortOrder, {
            id: 'created_on',
            desc: true
        })
    );
    const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>(initColumnFilter(search));
    const { data: appeals, isLoading } = useQuery({
        queryKey: ['appeals'],
        queryFn: async () => {
            return await apiGetAppeals({});
        }
    });

    const { Field, Subscribe, handleSubmit, reset, setFieldValue } = useForm({
        onSubmit: async ({ value }) => {
            setColumnFilters(
                initColumnFilter({
                    appeal_state: value.appeal_state ?? AppealState.Any,
                    source_id: value.source_id ?? undefined,
                    target_id: value.target_id ?? undefined
                })
            );
            await navigate({ to: '/admin/appeals', search: (prev) => ({ ...prev, ...value }) });
        },
        validatorAdapter: zodValidator,
        validators: {
            onChange: appealSearchSchema
        },
        defaultValues: {
            source_id: search.source_id ?? '',
            target_id: search.target_id ?? '',
            appeal_state: search.appeal_state ?? AppealState.Any
        }
    });

    const clear = async () => {
        //reset();
        setFieldValue('appeal_state', AppealState.Any);
        setFieldValue('source_id', '');
        setFieldValue('target_id', '');

        await handleSubmit();
        await navigate({
            to: '/admin/appeals',
            search: (prev) => ({ ...prev, source_id: undefined, target_id: undefined, appeal_state: undefined })
        });
    };

    return (
        <Grid container spacing={2}>
            <Title>Appeals</Title>
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
                            <Grid xs={6} md={4}>
                                <Field
                                    name={'source_id'}
                                    validators={makeSteamidValidatorsOptional()}
                                    children={(props) => {
                                        return (
                                            <TextFieldSimple {...props} label={'Author Steam ID'} fullwidth={true} />
                                        );
                                    }}
                                />
                            </Grid>

                            <Grid xs={6} md={4}>
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

                            <Grid xs={6} md={4}>
                                <Field
                                    name={'appeal_state'}
                                    children={(props) => {
                                        return (
                                            <SelectFieldSimple
                                                {...props}
                                                label={'Appeal Status'}
                                                fullwidth={true}
                                                items={AppealStateCollection}
                                                renderMenu={(item) => {
                                                    return (
                                                        <MenuItem value={item} key={`rs-${item}`}>
                                                            {appealStateString(item as AppealState)}
                                                        </MenuItem>
                                                    );
                                                }}
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
                <ContainerWithHeader title={'Recent Open Appeal Activity'}>
                    <AppealsTable
                        appeals={appeals ?? []}
                        isLoading={isLoading}
                        setColumnFilters={setColumnFilters}
                        columnFilters={columnFilters}
                        setSorting={setSorting}
                        sorting={sorting}
                        setPagination={setPagination}
                        pagination={pagination}
                    />
                </ContainerWithHeader>
            </Grid>
        </Grid>
        // </Formik>
    );
}
const columnHelper = createColumnHelper<SteamBanRecord>();

const AppealsTable = ({
    appeals,
    isLoading,
    setPagination,
    pagination,
    columnFilters,
    setColumnFilters,
    sorting,
    setSorting
}: { appeals: SteamBanRecord[]; isLoading: boolean } & TablePropsAll) => {
    const columns = [
        columnHelper.accessor('ban_id', {
            header: 'ID',
            size: 30,
            cell: (info) => (
                <TextLink
                    color={'primary'}
                    to={`/ban/$ban_id`}
                    params={{ ban_id: String(info.getValue()) }}
                    marginRight={2}
                >
                    #{info.getValue()}
                </TextLink>
            )
        }),
        columnHelper.accessor('appeal_state', {
            enableColumnFilter: true,
            filterFn: (row, _, value) => {
                if (value == AppealState.Any) {
                    return true;
                }
                return row.original.appeal_state == value;
            },
            header: 'Status',
            size: 85,
            cell: (info) => {
                return (
                    <TableCell>
                        <Typography variant={'body1'}>{appealStateString(info.getValue())}</Typography>
                    </TableCell>
                );
            }
        }),
        columnHelper.accessor('source_id', {
            enableColumnFilter: true,
            header: 'Author',
            size: 100,
            cell: (info) => (
                <PersonCell
                    showCopy={true}
                    steam_id={info.row.original.source_id}
                    personaname={info.row.original.source_personaname}
                    avatar_hash={info.row.original.source_avatarhash}
                />
            )
        }),
        columnHelper.accessor('target_id', {
            enableColumnFilter: true,
            header: 'Subject',
            size: 100,
            cell: (info) => (
                <PersonCell
                    showCopy={true}
                    steam_id={info.row.original.target_id}
                    personaname={info.row.original.target_personaname}
                    avatar_hash={info.row.original.target_avatarhash}
                />
            )
        }),
        columnHelper.accessor('reason', {
            header: 'Reason',
            size: 100,
            cell: (info) => <Typography>{BanReasons[info.getValue()]}</Typography>
        }),
        columnHelper.accessor('reason_text', {
            header: 'Custom',
            size: 150,
            cell: (info) => <Typography>{info.getValue()}</Typography>
        }),
        columnHelper.accessor('created_on', {
            header: 'Created',
            size: 120,
            cell: (info) => <Typography>{renderDateTime(info.getValue())}</Typography>
        }),
        columnHelper.accessor('updated_on', {
            header: 'Last Active',
            size: 120,
            cell: (info) => <Typography>{renderDateTime(info.getValue())}</Typography>
        })
    ];

    const table = useReactTable({
        data: appeals,
        columns: columns,
        autoResetPageIndex: true,
        getCoreRowModel: getCoreRowModel(),
        getFilteredRowModel: getFilteredRowModel(),
        getPaginationRowModel: getPaginationRowModel(),
        getSortedRowModel: getSortedRowModel(),
        onPaginationChange: setPagination,
        onSortingChange: setSorting,
        onColumnFiltersChange: setColumnFilters,

        state: {
            sorting,
            pagination,
            columnFilters
        }
    });

    return (
        <>
            <DataTable table={table} isLoading={isLoading} />{' '}
            <PaginatorLocal
                onRowsChange={(rows) => {
                    setPagination((prev) => {
                        return { ...prev, pageSize: rows };
                    });
                }}
                onPageChange={(page) => {
                    setPagination((prev) => {
                        return { ...prev, pageIndex: page };
                    });
                }}
                count={table.getRowCount()}
                rows={pagination.pageSize}
                page={pagination.pageIndex}
            />
        </>
    );
};
