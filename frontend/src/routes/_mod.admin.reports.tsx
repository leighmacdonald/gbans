import { useState } from 'react';
import FilterListIcon from '@mui/icons-material/FilterList';
import ReportIcon from '@mui/icons-material/Report';
import Link from '@mui/material/Link';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
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
    apiGetReports,
    BanReasons,
    ReportStatus,
    ReportStatusCollection,
    reportStatusString,
    ReportWithAuthor
} from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { DataTable } from '../component/DataTable.tsx';
import { PaginatorLocal } from '../component/PaginatorLocal.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import RouterLink from '../component/RouterLink.tsx';
import { TableHeadingCell } from '../component/TableHeadingCell.tsx';
import { Title } from '../component/Title';
import { Buttons } from '../component/field/Buttons.tsx';
import { SelectFieldSimple } from '../component/field/SelectFieldSimple.tsx';
import { TextFieldSimple } from '../component/field/TextFieldSimple.tsx';
import { initColumnFilter, initPagination, initSortOrder, TablePropsAll } from '../types/table.ts';
import { commonTableSearchSchema } from '../util/table.ts';
import { renderDateTime } from '../util/text.tsx';
import { makeSteamidValidatorsOptional } from '../util/validator/makeSteamidValidatorsOptional.ts';

const reportsSearchSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z
        .enum(['report_id', 'source_id', 'target_id', 'report_status', 'reason', 'created_on', 'updated_on'])
        .optional(),
    source_id: z.string().optional(),
    target_id: z.string().optional(),
    deleted: z.boolean().optional(),
    report_status: z.nativeEnum(ReportStatus).optional()
});

export const Route = createFileRoute('/_mod/admin/reports')({
    component: AdminReports,
    validateSearch: (search) => reportsSearchSchema.parse(search),
    loader: async ({ context, abortController }) => {
        const reports = await context.queryClient.fetchQuery({
            queryKey: ['adminReports'],
            queryFn: async () => {
                return apiGetReports({ deleted: false }, abortController);
            }
        });
        return reports ?? [];
    }
});

function AdminReports() {
    const navigate = useNavigate({ from: Route.fullPath });
    const { sortColumn, sortOrder, page, rows, source_id, target_id, report_status } = Route.useSearch();
    const reports = Route.useLoaderData();

    const [pagination, setPagination] = useState(initPagination(page, rows));
    const [sorting, setSorting] = useState<SortingState>(
        initSortOrder(sortColumn, sortOrder, {
            id: 'created_on',
            desc: true
        })
    );
    const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>(
        initColumnFilter({
            report_status: report_status ?? ReportStatus.Any,
            source_id: source_id ?? undefined,
            target_id: target_id ?? undefined
        })
    );

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            setColumnFilters(
                initColumnFilter({
                    report_status: value.report_status ?? ReportStatus.Any,
                    source_id: value.source_id ?? undefined,
                    target_id: value.target_id ?? undefined
                })
            );
            await navigate({ to: '/admin/reports', replace: true, search: (prev) => ({ ...prev, ...value }) });
        },
        validatorAdapter: zodValidator,
        validators: {
            onChange: reportsSearchSchema
        },
        defaultValues: {
            source_id: source_id ?? '',
            target_id: target_id ?? '',
            report_status: report_status ?? ReportStatus.Any
        }
    });

    const clear = async () => {
        await navigate({
            to: '/admin/reports',
            search: (prev) => ({ ...prev, source_id: undefined, target_id: undefined, report_status: undefined })
        });
        reset();
        await handleSubmit();
    };

    return (
        <Grid container spacing={2}>
            <Title>Reports</Title>
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
                                    validators={makeSteamidValidatorsOptional()}
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
                                    name={'report_status'}
                                    children={(props) => {
                                        return (
                                            <SelectFieldSimple
                                                {...props}
                                                label={'Report Status'}
                                                fullwidth={true}
                                                items={ReportStatusCollection}
                                                renderMenu={(item) => {
                                                    return (
                                                        <MenuItem value={item} key={`rs-${item}`}>
                                                            {reportStatusString(item as ReportStatus)}
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
                <ContainerWithHeader title={'Current User Reports'} iconLeft={<ReportIcon />}>
                    <ReportTable
                        reports={reports}
                        isLoading={false}
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
    );
}

const columnHelper = createColumnHelper<ReportWithAuthor>();

const ReportTable = ({
    reports,
    isLoading,
    setPagination,
    pagination,
    columnFilters,
    setColumnFilters,
    sorting,
    setSorting
}: { reports: ReportWithAuthor[]; isLoading: boolean } & TablePropsAll) => {
    const columns = [
        columnHelper.accessor('report_id', {
            enableColumnFilter: false,
            header: () => <TableHeadingCell name={'ID'} />,
            cell: (info) => (
                <Link
                    color={'primary'}
                    component={RouterLink}
                    to={`/report/$reportId`}
                    params={{ reportId: info.getValue() }}
                    marginRight={2}
                >
                    #{info.getValue()}
                </Link>
            )
        }),
        columnHelper.accessor('report_status', {
            filterFn: (row, _, value: ReportStatus) => {
                if (value == ReportStatus.Any) {
                    return true;
                }
                return row.original.report_status == value;
            },
            header: () => <TableHeadingCell name={'Status'} />,
            cell: (info) => {
                return (
                    <Stack direction={'row'} spacing={1}>
                        <Typography variant={'body1'}>{reportStatusString(info.getValue())}</Typography>
                    </Stack>
                );
            }
        }),
        columnHelper.accessor('source_id', {
            enableColumnFilter: true,
            header: () => <TableHeadingCell name={'Reporter'} />,
            cell: (info) => (
                <PersonCell
                    showCopy={true}
                    steam_id={info.row.original.author.steam_id}
                    personaname={info.row.original.author.personaname}
                    avatar_hash={info.row.original.author.avatarhash}
                />
            )
        }),
        columnHelper.accessor('target_id', {
            enableColumnFilter: true,
            header: () => <TableHeadingCell name={'Subject'} />,
            cell: (info) => (
                <PersonCell
                    showCopy={true}
                    steam_id={info.row.original.subject.steam_id}
                    personaname={info.row.original.subject.personaname}
                    avatar_hash={info.row.original.subject.avatarhash}
                />
            )
        }),
        columnHelper.accessor('reason', {
            enableColumnFilter: false,
            header: () => <TableHeadingCell name={'Reason'} />,
            cell: (info) => <Typography>{BanReasons[info.getValue()]}</Typography>
        }),
        columnHelper.accessor('created_on', {
            enableColumnFilter: false,
            header: () => <TableHeadingCell name={'Created'} />,
            cell: (info) => <Typography>{renderDateTime(info.getValue())}</Typography>
        }),
        columnHelper.accessor('updated_on', {
            enableColumnFilter: false,
            header: () => <TableHeadingCell name={'Updated'} />,
            cell: (info) => <Typography>{renderDateTime(info.getValue())}</Typography>
        })
    ];

    const table = useReactTable({
        data: reports,
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
            <DataTable table={table} isLoading={isLoading} />
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
