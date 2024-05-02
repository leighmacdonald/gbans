import WifiFindIcon from '@mui/icons-material/WifiFind';
import TableCell from '@mui/material/TableCell';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';
import { createColumnHelper, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import { z } from 'zod';
import { apiGetConnections, PersonConnection } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { DataTable, HeadingCell } from '../component/DataTable.tsx';
import { Paginator } from '../component/Paginator.tsx';
import { commonTableSearchSchema, LazyResult } from '../util/table.ts';
import { renderDateTime } from '../util/text.tsx';

const playersByIPSearchSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z.enum(['person_connection_id', 'steam_id', 'created_on', 'ip_addr', 'server_id']).catch('person_connection_id'),
    cidr: z.string().catch('')
});

export const Route = createFileRoute('/_mod/admin/network/playersbyip')({
    component: AdminNetworkPlayersByCIDR,
    validateSearch: (search) => playersByIPSearchSchema.parse(search)
});

function AdminNetworkPlayersByCIDR() {
    const { page, rows, sortOrder, sortColumn, cidr } = Route.useSearch();
    const { data: connections, isLoading } = useQuery({
        queryKey: ['playersByIP', { page, rows, sortOrder, sortColumn, cidr }],
        queryFn: async () => {
            if (cidr == '') {
                return { data: [], count: 0 };
            }
            return await apiGetConnections({
                limit: Number(rows),
                offset: Number((page ?? 0) * rows),
                order_by: sortColumn ?? 'steam_id',
                desc: sortOrder == 'desc',
                cidr: cidr
            });
        }
    });
    // const [state, setState] = useUrlState({
    //     page: undefined,
    //     source_id: undefined,
    //     asn: undefined,
    //     cidr: undefined,
    //     rows: undefined,
    //     sortOrder: undefined,
    //     sortColumn: undefined
    // });
    //
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
    // const onSubmit = (values: CIDRInputFieldProps) => {
    //     setState((prevState) => {
    //         return { ...prevState, cidr: values.cidr };
    //     });
    // };

    return (
        <ContainerWithHeader title={'Find Players By IP/CIDR'} iconLeft={<WifiFindIcon />}>
            <Grid container>
                <Grid xs={12}>
                    {/*<Formik onSubmit={onSubmit} initialValues={{ cidr: '' }}>*/}
                    <Grid container direction="row" alignItems="top" justifyContent="center" spacing={2}>
                        {/*<Grid xs>*/}
                        {/*    <NetworkRangeField />*/}
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
                    {/*        columns={[*/}
                    {/*            {*/}
                    {/*                label: 'Created',*/}
                    {/*                tooltip: 'Created On',*/}
                    {/*                sortKey: 'created_on',*/}
                    {/*                sortType: 'date',*/}
                    {/*                align: 'left',*/}
                    {/*                width: '150px',*/}
                    {/*                sortable: true,*/}
                    {/*                renderer: (obj: PersonConnection) => (*/}
                    {/*                    <Typography variant={'body1'}>*/}
                    {/*                        {renderDateTime(obj.created_on)}*/}
                    {/*                    </Typography>*/}
                    {/*                )*/}
                    {/*            },*/}
                    {/*            {*/}
                    {/*                label: 'Name',*/}
                    {/*                tooltip: 'Name',*/}
                    {/*                sortKey: 'persona_name',*/}
                    {/*                sortType: 'string',*/}
                    {/*                align: 'left',*/}
                    {/*                width: '200px',*/}
                    {/*                sortable: true*/}
                    {/*            },*/}
                    {/*            {*/}
                    {/*                label: 'SteamID',*/}
                    {/*                tooltip: 'Name',*/}
                    {/*                sortKey: 'steam_id',*/}
                    {/*                sortType: 'string',*/}
                    {/*                align: 'left',*/}
                    {/*                width: '200px',*/}
                    {/*                sortable: true*/}
                    {/*            },*/}
                    {/*            {*/}
                    {/*                label: 'IP Address',*/}
                    {/*                tooltip: 'IP Address',*/}
                    {/*                sortKey: 'ip_addr',*/}
                    {/*                sortType: 'string',*/}
                    {/*                align: 'left',*/}
                    {/*                width: '150px',*/}
                    {/*                sortable: true*/}
                    {/*            },*/}
                    {/*            {*/}
                    {/*                label: 'Server',*/}
                    {/*                tooltip: 'IP Address',*/}
                    {/*                sortKey: 'ip_addr',*/}
                    {/*                sortType: 'string',*/}
                    {/*                align: 'left',*/}
                    {/*                sortable: true,*/}
                    {/*                renderer: (obj: PersonConnection) => {*/}
                    {/*                    return (*/}
                    {/*                        <Tooltip*/}
                    {/*                            title={obj.server_name ?? 'Unknown'}*/}
                    {/*                        >*/}
                    {/*                            <Typography variant={'body1'}>*/}
                    {/*                                {obj.server_name_short ??*/}
                    {/*                                    'Unknown'}*/}
                    {/*                            </Typography>*/}
                    {/*                        </Tooltip>*/}
                    {/*                    );*/}
                    {/*                }*/}
                    {/*            }*/}
                    {/*        ]}*/}
                    {/*    />*/}
                    {/*)}*/}
                </Grid>
            </Grid>
            <PayersByIPTable connections={connections ?? { data: [], count: 0 }} isLoading={isLoading} />
            <Paginator page={page} rows={rows} data={connections} />
        </ContainerWithHeader>
    );
}

const columnHelper = createColumnHelper<PersonConnection>();

const PayersByIPTable = ({ connections, isLoading }: { connections: LazyResult<PersonConnection>; isLoading: boolean }) => {
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
        columnHelper.accessor('steam_id', {
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
