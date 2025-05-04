import { useMemo, useState } from 'react';
import FilterListIcon from '@mui/icons-material/FilterList';
import ReportIcon from '@mui/icons-material/Report';
import Grid from '@mui/material/Grid';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useForm } from '@tanstack/react-form';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { ColumnFiltersState, createColumnHelper, SortingState } from '@tanstack/react-table';
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
import { FullTable } from '../component/FullTable.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { TextLink } from '../component/TextLink.tsx';
import { Title } from '../component/Title';
import { Buttons } from '../component/field/Buttons.tsx';
import { SelectFieldSimple } from '../component/field/SelectFieldSimple.tsx';
import { TextFieldSimple } from '../component/field/TextFieldSimple.tsx';
import { initColumnFilter, initPagination, initSortOrder, makeCommonTableSearchSchema } from '../util/table.ts';
import { renderDateTime } from '../util/time.ts';

const reportsSearchSchema = z.object({
    ...makeCommonTableSearchSchema([
        'report_id',
        'source_id',
        'target_id',
        'report_status',
        'reason',
        'created_on',
        'updated_on'
    ]),
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
    const search = Route.useSearch();
    const reports = Route.useLoaderData();

    const [pagination, setPagination] = useState(initPagination(search.pageIndex, search.pageSize));
    const [sorting] = useState<SortingState>(
        initSortOrder(search.sortColumn, search.sortOrder, { id: 'report_id', desc: true })
    );
    const [columnFilters, setColumnFilters] = useState<ColumnFiltersState>(initColumnFilter(search));

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
        validators: {
            onSubmit: z.object({
                source_id: z.string(),
                target_id: z.string(),
                report_status: z.nativeEnum(ReportStatus)
            })
        },
        defaultValues: {
            source_id: search.source_id ?? '',
            target_id: search.target_id ?? '',
            report_status: search.report_status ?? ReportStatus.Any
        }
    });

    const clear = async () => {
        reset();
        setColumnFilters([]);
        await navigate({
            to: '/admin/reports',
            search: (prev) => ({ ...prev, source_id: undefined, target_id: undefined, report_status: undefined })
        });
    };

    const columns = useMemo(() => {
        return makeColumns();
    }, []);

    return (
        <Grid container spacing={2}>
            <Title>Reports</Title>
            <Grid size={{ xs: 12 }}>
                <ContainerWithHeader title={'Filters'} iconLeft={<FilterListIcon />} marginTop={2}>
                    <form
                        onSubmit={async (e) => {
                            e.preventDefault();
                            e.stopPropagation();
                            await handleSubmit();
                        }}
                    >
                        <Grid container spacing={2}>
                            <Grid size={{ xs: 6, md: 3 }}>
                                <Field
                                    name={'source_id'}
                                    children={(props) => {
                                        return (
                                            <TextFieldSimple
                                                {...props}
                                                value={props.state.value}
                                                label={'Author Steam ID'}
                                                fullwidth={true}
                                            />
                                        );
                                    }}
                                />
                            </Grid>

                            <Grid size={{ xs: 6, md: 3 }}>
                                <Field
                                    name={'target_id'}
                                    children={(props) => {
                                        return (
                                            <TextFieldSimple
                                                {...props}
                                                value={props.state.value}
                                                label={'Subject Steam ID'}
                                                fullwidth={true}
                                            />
                                        );
                                    }}
                                />
                            </Grid>

                            <Grid size={{ xs: 6, md: 3 }}>
                                <Field
                                    name={'report_status'}
                                    children={(props) => {
                                        return (
                                            <SelectFieldSimple
                                                {...props}
                                                value={props.state.value}
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
                            <Grid size={{ xs: 12 }}>
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
            <Grid size={{ xs: 12 }}>
                <ContainerWithHeader title={'Current User Reports'} iconLeft={<ReportIcon />}>
                    <FullTable
                        data={reports ?? []}
                        isLoading={false}
                        columns={columns}
                        sorting={sorting}
                        pagination={pagination}
                        setPagination={setPagination}
                        columnFilters={columnFilters}
                        toOptions={{ from: Route.fullPath }}
                    />
                </ContainerWithHeader>
            </Grid>
        </Grid>
    );
}

const columnHelper = createColumnHelper<ReportWithAuthor>();

const makeColumns = () => {
    return [
        columnHelper.accessor('report_id', {
            enableColumnFilter: false,
            header: 'ID',
            size: 30,
            cell: (info) => (
                <TextLink
                    color={'primary'}
                    to={`/report/$reportId`}
                    params={{ reportId: String(info.getValue()) }}
                    marginRight={2}
                >
                    #{info.getValue()}
                </TextLink>
            )
        }),
        columnHelper.accessor('report_status', {
            size: 120,
            filterFn: (row, _, value: ReportStatus) => {
                if (value == ReportStatus.Any) {
                    return true;
                }
                return row.original.report_status == value;
            },
            header: 'Status',
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
            header: 'Reporter',
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
            header: 'Subject',
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
            header: 'Reason',
            size: 100,
            cell: (info) => <Typography>{BanReasons[info.getValue()]}</Typography>
        }),
        columnHelper.accessor('created_on', {
            enableColumnFilter: false,
            size: 100,
            header: 'Created',
            cell: (info) => <Typography>{renderDateTime(info.getValue())}</Typography>
        }),
        columnHelper.accessor('updated_on', {
            enableColumnFilter: false,
            size: 100,
            header: 'Updated',
            cell: (info) => <Typography>{renderDateTime(info.getValue())}</Typography>
        })
    ];
};
