import FilterListIcon from '@mui/icons-material/FilterList';
import ReportIcon from '@mui/icons-material/Report';
import VisibilityIcon from '@mui/icons-material/Visibility';
import ButtonGroup from '@mui/material/ButtonGroup';
import IconButton from '@mui/material/IconButton';
import MenuItem from '@mui/material/MenuItem';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute, Link as RouterLink, useNavigate } from '@tanstack/react-router';
import { createColumnHelper, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import { zodValidator } from '@tanstack/zod-form-adapter';
import { z } from 'zod';
import { apiGetReports, BanReasons, ReportStatus, ReportStatusCollection, reportStatusString, ReportWithAuthor } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { DataTable, HeadingCell } from '../component/DataTable.tsx';
import { Paginator } from '../component/Paginator.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { ReportStatusIcon } from '../component/ReportStatusIcon.tsx';
import { Buttons } from '../component/field/Buttons.tsx';
import { SelectFieldSimple } from '../component/field/SelectFieldSimple.tsx';
import { makeSteamidValidatorsOptional } from '../component/field/SteamIDField.tsx';
import { TextFieldSimple } from '../component/field/TextFieldSimple.tsx';
import { commonTableSearchSchema, LazyResult, RowsPerPage } from '../util/table.ts';
import { renderDateTime } from '../util/text.tsx';

const reportsSearchSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z.enum(['report_id', 'source_id', 'target_id', 'report_status', 'reason', 'created_on', 'updated_on']).optional(),
    source_id: z.string().optional(),
    target_id: z.string().optional(),
    deleted: z.boolean().optional(),
    report_status: z.nativeEnum(ReportStatus).optional()
});

export const Route = createFileRoute('/_mod/admin/reports')({
    component: AdminReports,
    validateSearch: (search) => reportsSearchSchema.parse(search)
});

function AdminReports() {
    const defaultRows = RowsPerPage.TwentyFive;
    const navigate = useNavigate({ from: Route.fullPath });
    const { page, sortColumn, rows, sortOrder, source_id, target_id, report_status } = Route.useSearch();
    const { data: reports, isLoading } = useQuery({
        queryKey: ['reports', { page, rows, sortOrder, sortColumn, source_id, target_id, report_status }],
        queryFn: async () => {
            return apiGetReports({
                limit: rows ?? defaultRows,
                offset: (page ?? 0) * (rows ?? defaultRows),
                order_by: sortColumn ?? 'report_id',
                desc: sortOrder == 'desc',
                source_id: source_id,
                target_id: target_id,
                report_status: Number(report_status)
            });
        }
    });

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            await navigate({ to: '/admin/reports', search: (prev) => ({ ...prev, ...value }) });
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
    };

    return (
        <Grid container spacing={2}>
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
                                        return <TextFieldSimple {...props} label={'Author Steam ID'} fullwidth={true} />;
                                    }}
                                />
                            </Grid>

                            <Grid xs={6} md={3}>
                                <Field
                                    name={'target_id'}
                                    validators={makeSteamidValidatorsOptional()}
                                    children={(props) => {
                                        return <TextFieldSimple {...props} label={'Subject Steam ID'} fullwidth={true} />;
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
                                        <Buttons reset={reset} canSubmit={canSubmit} isSubmitting={isSubmitting} onClear={clear} />
                                    )}
                                />
                            </Grid>
                        </Grid>
                    </form>
                </ContainerWithHeader>
            </Grid>
            <Grid xs={12}>
                <ContainerWithHeader title={'Current User Reports'} iconLeft={<ReportIcon />}>
                    <ReportTable reports={reports ?? { data: [], count: 0 }} isLoading={isLoading} />
                    <Paginator data={reports} page={page ?? 0} rows={rows ?? defaultRows} path={'/admin/reports'} />
                </ContainerWithHeader>
            </Grid>
        </Grid>
        // </Formik>
    );
}

const columnHelper = createColumnHelper<ReportWithAuthor>();

const ReportTable = ({ reports, isLoading }: { reports: LazyResult<ReportWithAuthor>; isLoading: boolean }) => {
    const columns = [
        columnHelper.accessor('report_id', {
            header: () => <HeadingCell name={'View'} />,
            cell: (info) => (
                <ButtonGroup>
                    <IconButton color={'primary'} component={RouterLink} to={`/report/$reportId`} params={{ reportId: info.getValue() }}>
                        <Tooltip title={'View'}>
                            <VisibilityIcon />
                        </Tooltip>
                    </IconButton>
                </ButtonGroup>
            )
        }),
        columnHelper.accessor('report_status', {
            header: () => <HeadingCell name={'Status'} />,
            cell: (info) => {
                return (
                    <Stack direction={'row'} spacing={1}>
                        <ReportStatusIcon reportStatus={info.getValue()} />
                        <Typography variant={'body1'}>{reportStatusString(info.getValue())}</Typography>
                    </Stack>
                );
            }
        }),
        columnHelper.accessor('source_id', {
            header: () => <HeadingCell name={'Reporter'} />,
            cell: (info) => (
                <PersonCell
                    steam_id={reports.data[info.row.index].author.steam_id}
                    personaname={reports.data[info.row.index].author.personaname}
                    avatar_hash={reports.data[info.row.index].author.avatarhash}
                />
            )
        }),
        columnHelper.accessor('subject', {
            header: () => <HeadingCell name={'Subject'} />,
            cell: (info) => (
                <PersonCell
                    steam_id={reports.data[info.row.index].subject.steam_id}
                    personaname={reports.data[info.row.index].subject.personaname}
                    avatar_hash={reports.data[info.row.index].subject.avatarhash}
                />
            )
        }),
        columnHelper.accessor('reason', {
            header: () => <HeadingCell name={'Reason'} />,
            cell: (info) => <Typography>{BanReasons[info.getValue()]}</Typography>
        }),
        columnHelper.accessor('created_on', {
            header: () => <HeadingCell name={'Created'} />,
            cell: (info) => <Typography>{renderDateTime(info.getValue())}</Typography>
        }),
        columnHelper.accessor('updated_on', {
            header: () => <HeadingCell name={'Updated'} />,
            cell: (info) => <Typography>{renderDateTime(info.getValue())}</Typography>
        })
    ];

    const table = useReactTable({
        data: reports.data,
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        manualPagination: true,
        autoResetPageIndex: true
    });

    return <DataTable table={table} isLoading={isLoading} />;
};
