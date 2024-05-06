import FilterListIcon from '@mui/icons-material/FilterList';
import WifiFindIcon from '@mui/icons-material/WifiFind';
import Link from '@mui/material/Link';
import TableCell from '@mui/material/TableCell';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useForm } from '@tanstack/react-form';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { createColumnHelper, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import { zodValidator } from '@tanstack/zod-form-adapter';
import { z } from 'zod';
import { apiGetConnections, PersonConnection } from '../api';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { DataTable } from '../component/DataTable.tsx';
import { Paginator } from '../component/Paginator.tsx';
import RouterLink from '../component/RouterLink.tsx';
import { TableHeadingCell } from '../component/TableHeadingCell.tsx';
import { Buttons } from '../component/field/Buttons.tsx';
import { TextFieldSimple } from '../component/field/TextFieldSimple.tsx';
import { commonTableSearchSchema, LazyResult, RowsPerPage } from '../util/table.ts';
import { renderDateTime } from '../util/text.tsx';
import { emptyOrNullString } from '../util/types.ts';

const playersByIPSearchSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z.enum(['person_connection_id', 'steam_id', 'created_on', 'ip_addr', 'server_id']).optional(),
    cidr: z.string().optional()
});

export const Route = createFileRoute('/_mod/admin/network/playersbyip')({
    component: AdminNetworkPlayersByCIDR,
    validateSearch: (search) => playersByIPSearchSchema.parse(search)
});

function AdminNetworkPlayersByCIDR() {
    const defaultRows = RowsPerPage.TwentyFive;
    const navigate = useNavigate({ from: Route.fullPath });
    const { page, rows, sortOrder, sortColumn, cidr } = Route.useSearch();
    const { data: connections, isLoading } = useQuery({
        queryKey: ['playersByIP', { page, rows, sortOrder, sortColumn, cidr }],
        queryFn: async () => {
            if (emptyOrNullString(cidr)) {
                return { data: [], count: 0 };
            }
            return await apiGetConnections({
                limit: rows ?? defaultRows,
                offset: (page ?? 0) * (rows ?? defaultRows),
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

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            await navigate({ to: '/admin/network/playersbyip', search: (prev) => ({ ...prev, ...value }) });
        },
        validatorAdapter: zodValidator,
        validators: {
            onChange: playersByIPSearchSchema
        },
        defaultValues: {
            cidr: cidr ?? ''
        }
    });

    const clear = async () => {
        await navigate({
            to: '/admin/network/playersbyip',
            search: (prev) => ({ ...prev, source_id: undefined })
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
                            <Grid xs={12}>
                                <Field
                                    name={'cidr'}
                                    children={(props) => {
                                        return <TextFieldSimple {...props} fullwidth={true} label={'CIDR/IP'} />;
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
                <ContainerWithHeader title={'Find Players By IP/CIDR'} iconLeft={<WifiFindIcon />}>
                    <PayersByIPTable connections={connections ?? { data: [], count: 0 }} isLoading={isLoading} />
                    <Paginator page={page ?? 0} rows={rows ?? defaultRows} data={connections} path={'/admin/network/playersbyip'} />
                </ContainerWithHeader>
            </Grid>
        </Grid>
    );
}

const columnHelper = createColumnHelper<PersonConnection>();

const PayersByIPTable = ({ connections, isLoading }: { connections: LazyResult<PersonConnection>; isLoading: boolean }) => {
    const columns = [
        columnHelper.accessor('created_on', {
            header: () => <TableHeadingCell name={'Created'} />,
            cell: (info) => <Typography>{renderDateTime(info.getValue())}</Typography>
        }),
        columnHelper.accessor('persona_name', {
            header: () => <TableHeadingCell name={'Name'} />,
            cell: (info) => (
                <TableCell>
                    <Typography>{info.getValue()}</Typography>
                </TableCell>
            )
        }),
        columnHelper.accessor('steam_id', {
            header: () => <TableHeadingCell name={'Steam ID'} />,
            cell: (info) => (
                <TableCell>
                    <Link component={RouterLink} to={'/profile/$steamId'} params={{ steamId: info.getValue() }}>
                        {info.getValue()}
                    </Link>
                </TableCell>
            )
        }),
        columnHelper.accessor('ip_addr', {
            header: () => <TableHeadingCell name={'IP Address'} />,
            cell: (info) => (
                <TableCell>
                    <Typography>{info.getValue()}</Typography>
                </TableCell>
            )
        }),
        columnHelper.accessor('server_id', {
            header: () => <TableHeadingCell name={'Server'} />,
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
