import FilterListIcon from '@mui/icons-material/FilterList';
import ReportIcon from '@mui/icons-material/Report';
import VisibilityIcon from '@mui/icons-material/Visibility';
import ButtonGroup from '@mui/material/ButtonGroup';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute, Link as RouterLink } from '@tanstack/react-router';
import { createColumnHelper, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import { z } from 'zod';
import { apiGetReports, BanReasons, ReportStatus, reportStatusString, ReportWithAuthor } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { DataTable, HeadingCell } from '../component/DataTable.tsx';
import { Paginator } from '../component/Paginator.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { ReportStatusIcon } from '../component/ReportStatusIcon.tsx';
import { commonTableSearchSchema, LazyResult } from '../util/table.ts';
import { renderDateTime } from '../util/text.tsx';

const reportsSearchSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z.enum(['report_id', 'source_id', 'target_id', 'report_status', 'reason', 'created_on', 'updated_on']).catch('updated_on'),
    source_id: z.string().catch(''),
    target_id: z.string().catch(''),
    deleted: z.boolean().catch(false),
    report_status: z.nativeEnum(ReportStatus).catch(ReportStatus.Any)
});

export const Route = createFileRoute('/_mod/admin/reports')({
    component: AdminReports,
    validateSearch: (search) => reportsSearchSchema.parse(search)
});

function AdminReports() {
    const { page, sortColumn, rows, sortOrder, source_id, target_id, report_status } = Route.useSearch();
    const { data: reports, isLoading } = useQuery({
        queryKey: ['reports', { page, rows, sortOrder, sortColumn, source_id, target_id, report_status }],
        queryFn: async () => {
            return apiGetReports({
                limit: Number(rows),
                offset: Number((page ?? 0) * rows),
                order_by: sortColumn ?? 'report_id',
                desc: sortOrder == 'desc',
                source_id: source_id,
                target_id: target_id,
                report_status: Number(report_status)
            });
        }
    });
    //
    // const onFilterSubmit = useCallback(
    //     (values: FilterValues) => {
    //         setState({
    //             reportStatus: values.report_status != ReportStatus.Any ? values.report_status : undefined,
    //             source: values.source_id != '' ? values.source_id : undefined,
    //             target: values.target_id != '' ? values.target_id : undefined
    //         });
    //     },
    //     [setState]
    // );
    //
    // const onFilterReset = useCallback(() => {
    //     setState({
    //         reportStatus: undefined,
    //         source: undefined,
    //         target: undefined
    //     });
    // }, [setState]);

    return (
        // <Formik<FilterValues>
        //     onSubmit={onFilterSubmit}
        //     onReset={onFilterReset}
        //     initialValues={{
        //         report_status: Number(state.reportStatus ?? ReportStatus.Any),
        //         source_id: state.source,
        //         target_id: state.target
        //     }}
        //     validationSchema={validationSchema}
        //     validateOnChange={true}
        //     validateOnBlur={true}
        // >
        <Grid container spacing={2}>
            <Grid xs={12}>
                <ContainerWithHeader title={'Filters'} iconLeft={<FilterListIcon />}>
                    <Grid container spacing={2}>
                        {/*<Grid xs={4} sm={4} md={3}>*/}
                        {/*    <SourceIDField />*/}
                        {/*</Grid>*/}
                        {/*<Grid xs={4} sm={4} md={3}>*/}
                        {/*    <TargetIDField />*/}
                        {/*</Grid>*/}
                        {/*<Grid xs={4} sm={4} md={3}>*/}
                        {/*    <ReportStatusField />*/}
                        {/*</Grid>*/}
                        {/*<Grid xs={4} sm={4} md={3}>*/}
                        {/*    <FilterButtons />*/}
                        {/*</Grid>*/}
                    </Grid>
                </ContainerWithHeader>
            </Grid>
            <Grid xs={12}>
                <ContainerWithHeader title={'Current User Reports'} iconLeft={<ReportIcon />}>
                    <ReportTable reports={reports ?? { data: [], count: 0 }} isLoading={isLoading} />
                    <Paginator data={reports} page={page} rows={rows} />
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
