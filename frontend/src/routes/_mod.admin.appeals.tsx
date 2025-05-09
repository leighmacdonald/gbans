import { useState } from 'react';
import FilterListIcon from '@mui/icons-material/FilterList';
import Grid from '@mui/material/Grid';
import MenuItem from '@mui/material/MenuItem';
import Typography from '@mui/material/Typography';
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
import { useAppForm } from '../contexts/formContext.tsx';
import { TablePropsAll } from '../types/table.ts';
import { commonTableSearchSchema, initColumnFilter, initPagination, initSortOrder } from '../util/table.ts';
import { renderDateTime } from '../util/time.ts';
import { makeValidateSteamIDCallback } from '../util/validator/makeValidateSteamIDCallback.ts';

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

    const form = useAppForm({
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
        validators: {
            onChangeAsyncDebounceMs: 500,
            onChangeAsync: z.object({
                source_id: makeValidateSteamIDCallback(),
                target_id: makeValidateSteamIDCallback(),
                appeal_state: z.nativeEnum(AppealState)
            })
        },
        defaultValues: {
            source_id: search.source_id ?? '',
            target_id: search.target_id ?? '',
            appeal_state: search.appeal_state ?? AppealState.Any
        }
    });

    const clear = async () => {
        //reset();
        form.setFieldValue('appeal_state', AppealState.Any);
        form.setFieldValue('source_id', '');
        form.setFieldValue('target_id', '');

        await form.handleSubmit();
        await navigate({
            to: '/admin/appeals',
            search: (prev) => ({ ...prev, source_id: undefined, target_id: undefined, appeal_state: undefined })
        });
    };

    return (
        <Grid container spacing={2}>
            <Title>Appeals</Title>
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
                            <Grid size={{ xs: 6, md: 4 }}>
                                <form.AppField
                                    name={'source_id'}
                                    children={(field) => {
                                        return <field.TextField label={'Author Steam ID'} />;
                                    }}
                                />
                            </Grid>

                            <Grid size={{ xs: 6, md: 4 }}>
                                <form.AppField
                                    name={'target_id'}
                                    children={(field) => {
                                        return <field.TextField label={'Subject Steam ID'} />;
                                    }}
                                />
                            </Grid>

                            <Grid size={{ xs: 6, md: 4 }}>
                                <form.AppField
                                    name={'appeal_state'}
                                    children={(field) => {
                                        return (
                                            <field.SelectField
                                                label={'Appeal Status'}
                                                items={AppealStateCollection}
                                                renderItem={(item) => {
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
                            <Grid size={{ xs: 12 }}>
                                <form.AppForm>
                                    <form.ResetButton onClick={clear} />
                                    <form.SubmitButton />
                                </form.AppForm>
                            </Grid>
                        </Grid>
                    </form>
                </ContainerWithHeader>
            </Grid>

            <Grid size={{ xs: 12 }}>
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
                return <Typography variant={'body1'}>{appealStateString(info.getValue())}</Typography>;
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
            size: 150,
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
