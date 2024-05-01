import FilterListIcon from '@mui/icons-material/FilterList';
import VisibilityIcon from '@mui/icons-material/Visibility';
import Link from '@mui/material/Link';
import TableCell from '@mui/material/TableCell';
import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute, Link as RouterLink } from '@tanstack/react-router';
import { createColumnHelper, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import { z } from 'zod';
import { apiGetAppeals, AppealState, appealStateString, BanReasons, SteamBanRecord } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { DataTable, HeadingCell } from '../component/DataTable.tsx';
import { Paginator } from '../component/Paginator.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { commonTableSearchSchema, LazyResult, RowsPerPage } from '../util/table.ts';
import { renderDateTime } from '../util/text.tsx';

const appealSearchSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z.enum(['report_id', 'source_id', 'target_id', 'appeal_state', 'reason', 'created_on', 'updated_on']).catch('updated_on'),
    source_id: z.string().catch(''),
    target_id: z.string().catch(''),
    appeal_state: z.nativeEnum(AppealState).catch(AppealState.Any)
});

export const Route = createFileRoute('/_mod/admin/appeals')({
    component: AdminAppeals,
    validateSearch: (search) => appealSearchSchema.parse(search)
});

function AdminAppeals() {
    const { page, sortColumn, rows, sortOrder, source_id, target_id, appeal_state } = Route.useSearch();

    const { data: appeals, isLoading } = useQuery({
        queryKey: ['appeals'],
        queryFn: async () => {
            return await apiGetAppeals({
                limit: Number(rows ?? RowsPerPage.TwentyFive),
                offset: Number((page ?? 0) * (rows ?? RowsPerPage.TwentyFive)),
                order_by: sortColumn ?? 'ban_id',
                desc: (sortOrder ?? 'desc') == 'desc',
                source_id: source_id ?? '',
                target_id: target_id ?? '',
                appeal_state: Number(appeal_state ?? AppealState.Any)
            });
        }
    });

    // const tableIcon = useMemo(() => {
    //     if (loading) {
    //         return <LoadingSpinner />;
    //     }
    //     switch (state.appealState) {
    //         case AppealState.Accepted:
    //             return <GppGoodIcon />;
    //         case AppealState.Open:
    //             return <FiberNewIcon />;
    //         case AppealState.Denied:
    //             return <DoNotDisturbIcon />;
    //         default:
    //             return <SnoozeIcon />;
    //     }
    // }, [loading, state.appealState]);
    //
    // const onSubmit = useCallback(
    //     (values: AppealFilterValues) => {
    //         setState({
    //             appealState: values.appeal_state != AppealState.Any ? values.appeal_state : undefined,
    //             source: values.source_id != '' ? values.source_id : undefined,
    //             target: values.target_id != '' ? values.target_id : undefined
    //         });
    //     },
    //     [setState]
    // );
    //
    // const onReset = useCallback(() => {
    //     setState({
    //         appealState: undefined,
    //         source: undefined,
    //         target: undefined
    //     });
    // }, [setState]);

    return (
        // <Formik<AppealFilterValues>
        //     initialValues={{
        //         appeal_state: Number(state.appealState ?? AppealState.Any),
        //         source_id: state.source,
        //         target_id: state.target
        //     }}
        //     onReset={onReset}
        //     onSubmit={onSubmit}
        //     validationSchema={validationSchema}
        //     validateOnChange={true}
        // >
        <Grid container spacing={3}>
            <Grid xs={12}>
                <ContainerWithHeader title={'Appeal Activity Filters'} iconLeft={<FilterListIcon />}>
                    <Grid container spacing={2}>
                        {/*<Grid xs={6} sm={4} md={3}>*/}
                        {/*    <AppealStateField />*/}
                        {/*</Grid>*/}
                        {/*<Grid xs={6} sm={4} md={3}>*/}
                        {/*    <SourceIDField />*/}
                        {/*</Grid>*/}
                        {/*<Grid xs={6} sm={4} md={3}>*/}
                        {/*    <TargetIDField />*/}
                        {/*</Grid>*/}
                        {/*<Grid xs={6} sm={4} md={3}>*/}
                        {/*    <FilterButtons />*/}
                        {/*</Grid>*/}
                    </Grid>
                </ContainerWithHeader>
            </Grid>

            <Grid xs={12}>
                <ContainerWithHeader title={'Recent Open Appeal Activity'}>
                    <AppealsTable appeals={appeals ?? { data: [], count: 0 }} isLoading={isLoading} />
                    <Paginator page={page} rows={rows} />
                </ContainerWithHeader>
            </Grid>
        </Grid>
        // </Formik>
    );
}
const columnHelper = createColumnHelper<SteamBanRecord>();

const AppealsTable = ({ appeals, isLoading }: { appeals: LazyResult<SteamBanRecord>; isLoading: boolean }) => {
    const columns = [
        columnHelper.accessor('ban_id', {
            header: () => <HeadingCell name={'View'} />,
            cell: (info) => (
                <Link color={'primary'} component={RouterLink} to={`/ban/$ban_id`} params={{ ban_id: info.getValue() }}>
                    <Tooltip title={'View'}>
                        <VisibilityIcon />
                    </Tooltip>
                </Link>
            )
        }),
        columnHelper.accessor('appeal_state', {
            header: () => <HeadingCell name={'Status'} />,
            cell: (info) => {
                return (
                    <TableCell>
                        <Typography variant={'body1'}>{appealStateString(info.getValue())}</Typography>
                    </TableCell>
                );
            }
        }),
        columnHelper.accessor('source_id', {
            header: () => <HeadingCell name={'Author'} />,
            cell: (info) => (
                <PersonCell
                    steam_id={appeals.data[info.row.index].source_id}
                    personaname={appeals.data[info.row.index].source_personaname}
                    avatar_hash={appeals.data[info.row.index].source_avatarhash}
                />
            )
        }),
        columnHelper.accessor('target_id', {
            header: () => <HeadingCell name={'Subject'} />,
            cell: (info) => (
                <PersonCell
                    steam_id={appeals.data[info.row.index].target_id}
                    personaname={appeals.data[info.row.index].target_personaname}
                    avatar_hash={appeals.data[info.row.index].target_avatarhash}
                />
            )
        }),
        columnHelper.accessor('reason', {
            header: () => <HeadingCell name={'Reason'} />,
            cell: (info) => <Typography>{BanReasons[info.getValue()]}</Typography>
        }),
        columnHelper.accessor('reason_text', {
            header: () => <HeadingCell name={'Custom Reason'} />,
            cell: (info) => <Typography>{info.getValue()}</Typography>
        }),
        columnHelper.accessor('created_on', {
            header: () => <HeadingCell name={'Created'} />,
            cell: (info) => <Typography>{renderDateTime(info.getValue())}</Typography>
        }),
        columnHelper.accessor('updated_on', {
            header: () => <HeadingCell name={'Last Active'} />,
            cell: (info) => <Typography>{renderDateTime(info.getValue())}</Typography>
        })
    ];

    const table = useReactTable({
        data: appeals.data,
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        manualPagination: true,
        autoResetPageIndex: true
    });

    return <DataTable table={table} isLoading={isLoading} />;
};
