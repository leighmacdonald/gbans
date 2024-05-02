import SensorOccupiedIcon from '@mui/icons-material/SensorOccupied';
import TableCell from '@mui/material/TableCell';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';
import { createColumnHelper, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import { z } from 'zod';
import { apiGetConnections, PersonConnection } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { DataTable, HeadingCell } from '../component/DataTable.tsx';
import { Paginator } from '../component/Paginator.tsx';
import { commonTableSearchSchema, LazyResult } from '../util/table.ts';
import { renderDateTime } from '../util/text.tsx';

const ipHistorySearchSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z.enum(['person_connection_id', 'steam_id', 'created_on', 'ip_addr', 'server_id']).catch('person_connection_id'),
    source_id: z.string().catch('')
});

export const Route = createFileRoute('/_mod/admin/network/iphist')({
    component: AdminNetworkPlayerIPHistory,
    validateSearch: (search) => ipHistorySearchSchema.parse(search)
});

function AdminNetworkPlayerIPHistory() {
    const { page, rows, sortOrder, sortColumn, source_id } = Route.useSearch();
    const { data: connections, isLoading } = useQuery({
        queryKey: ['connectionHist', { page, rows, sortOrder, sortColumn, source_id }],
        queryFn: async () => {
            if (source_id == '') {
                return { data: [], count: 0 };
            }
            return await apiGetConnections({
                limit: Number(rows),
                offset: Number((page ?? 0) * rows),
                order_by: sortColumn ?? 'steam_id',
                desc: sortOrder == 'desc',
                source_id: source_id
            });
        }
    });
    // const {
    //     data: rows,
    //     count,
    //     loading
    // } = useConnections({
    //     limit: state.rows ?? RowsPerPage.TwentyFive,
    //     offset: Number((state.page ?? 0) * (state.rows ?? RowsPerPage.Ten)),
    //     order_by: state.sortColumn ?? 'created_on',
    //     desc: (state.sortOrder ?? 'desc') == 'desc',
    //     source_id: state.source_id ?? '',
    //     asn: 0,
    //     cidr: state.cidr ?? ''
    // });
    //
    // const onSubmit = (values: SourceIDFieldValue) => {
    //     setState((prevState) => {
    //         return { ...prevState, source_id: values.source_id };
    //     });
    // };

    return (
        <ContainerWithHeader title="Player IP History" iconLeft={<SensorOccupiedIcon />}>
            <Grid container spacing={2}>
                <Grid xs={12}>
                    {/*<Formik onSubmit={onSubmit} initialValues={{ source_id: '' }}>*/}
                    <Grid container direction="row" alignItems="top" justifyContent="center" spacing={2}>
                        {/*<Grid xs>*/}
                        {/*    <SourceIDField />*/}
                        {/*</Grid>*/}
                        {/*<Grid xs={2}>*/}
                        {/*    <SubmitButton*/}
                        {/*        label={'Submit'}*/}
                        {/*        fullWidth*/}
                        {/*        disabled={loading}*/}
                        {/*        startIcon={<SearchIcon />}*/}
                        {/*    />*/}
                        {/*</Grid>*/}
                    </Grid>
                    {/*</Formik>*/}
                </Grid>
                <Grid xs={12}>
                    {/*{loading ? (*/}
                    {/*    <LoadingPlaceholder />*/}
                    {/*) : (*/}
                    {/*    <LazyTable<PersonConnection>*/}
                    {/*        showPager={true}*/}
                    {/*        count={count}*/}
                    {/*        rows={rows}*/}
                    {/*        page={state.page}*/}
                    {/*        rowsPerPage={state.rows}*/}
                    {/*        sortOrder={state.sortOrder}*/}
                    {/*        sortColumn={state.sortColumn}*/}
                    {/*        onSortColumnChanged={async (column) => {*/}
                    {/*            setState((prevState) => {*/}
                    {/*                return { ...prevState, sortColumn: column };*/}
                    {/*            });*/}
                    {/*        }}*/}
                    {/*        onSortOrderChanged={async (direction) => {*/}
                    {/*            setState((prevState) => {*/}
                    {/*                return { ...prevState, sortOrder: direction };*/}
                    {/*            });*/}
                    {/*        }}*/}
                    {/*        onPageChange={(_, newPage: number) => {*/}
                    {/*            setState((prevState) => {*/}
                    {/*                return { ...prevState, page: newPage };*/}
                    {/*            });*/}
                    {/*        }}*/}
                    {/*        onRowsPerPageChange={(*/}
                    {/*            event: ChangeEvent<*/}
                    {/*                HTMLInputElement | HTMLTextAreaElement*/}
                    {/*            >*/}
                    {/*        ) => {*/}
                    {/*            setState((prevState) => {*/}
                    {/*                return {*/}
                    {/*                    ...prevState,*/}
                    {/*                    rows: parseInt(event.target.value, 10),*/}
                    {/*                    page: 0*/}
                    {/*                };*/}
                    {/*            });*/}
                    {/*        }}*/}
                    {/*        columns={connectionColumns}*/}
                    {/*    />*/}
                    {/*)}*/}
                    <IPHistoryTable connections={connections ?? { data: [], count: 0 }} isLoading={isLoading} />
                    <Paginator page={page} rows={rows} data={connections} />
                </Grid>
            </Grid>
        </ContainerWithHeader>
    );
}

const columnHelper = createColumnHelper<PersonConnection>();

const IPHistoryTable = ({ connections, isLoading }: { connections: LazyResult<PersonConnection>; isLoading: boolean }) => {
    const columns = [
        columnHelper.accessor('created_on', {
            header: () => <HeadingCell name={'Created'} />,
            cell: (info) => <Typography>{renderDateTime(info.getValue())}</Typography>
        }),
        columnHelper.accessor('persona_name', {
            header: () => <HeadingCell name={'Name'} />,
            cell: (info) => (
                <TableCell>
                    <Typography>{info.getValue()}</Typography>
                </TableCell>
            )
        }),
        columnHelper.accessor('ip_addr', {
            header: () => <HeadingCell name={'IP Address'} />,
            cell: (info) => (
                <TableCell>
                    <Typography>{info.getValue()}</Typography>
                </TableCell>
            )
        }),
        columnHelper.accessor('server_id', {
            header: () => <HeadingCell name={'Server'} />,
            cell: (info) => (
                <TableCell>
                    <Typography>{connections.data[info.row.index].server_name_short}</Typography>
                </TableCell>
            )
        })
    ];

    const table = useReactTable({
        data: connections.data,
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        manualPagination: true,
        autoResetPageIndex: true
    });

    return <DataTable table={table} isLoading={isLoading} />;
};
