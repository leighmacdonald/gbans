import AddIcon from '@mui/icons-material/Add';
import StorageIcon from '@mui/icons-material/Storage';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import Grid from '@mui/material/Unstable_Grid2';
import { useQuery } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';
import { createColumnHelper, getCoreRowModel, useReactTable } from '@tanstack/react-table';
import { z } from 'zod';
import { apiGetServersAdmin, Server } from '../api';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { DataTable } from '../component/DataTable.tsx';
import { Paginator } from '../component/Paginator.tsx';
import { TableCellBool } from '../component/TableCellBool.tsx';
import { TableHeadingCell } from '../component/TableHeadingCell.tsx';
import { commonTableSearchSchema, LazyResult, RowsPerPage } from '../util/table.ts';

const serversSearchSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z.enum(['server_id', 'short_name', 'name', 'address', 'port', 'region', 'cc', 'enable_stats', 'is_enabled']).optional()
});

export const Route = createFileRoute('/_admin/admin/servers')({
    component: AdminServers,
    validateSearch: (search) => serversSearchSchema.parse(search)
});

function AdminServers() {
    const defaultRows = RowsPerPage.TwentyFive;
    const { page, sortColumn, rows, sortOrder } = Route.useSearch();

    const { data: servers, isLoading } = useQuery({
        queryKey: ['serversAdmin', { page, sortColumn, sortOrder, rows }],
        queryFn: async () => {
            return await apiGetServersAdmin({
                limit: rows ?? defaultRows,
                offset: (page ?? 0) * (rows ?? defaultRows),
                order_by: sortColumn ?? 'name',
                desc: sortOrder == 'desc',
                include_disabled: false
            });
        }
    });
    // const [newServers, setNewServers] = useState<Server[]>([]);
    //
    // const { sendFlash } = useUserFlashCtx();
    // const { data, count } = useServersAdmin({
    //     limit: Number(state.rows ?? RowsPerPage.TwentyFive),
    //     offset: Number((state.page ?? 0) * (state.rows ?? RowsPerPage.TwentyFive)),
    //     order_by: state.sortColumn ?? 'short_name',
    //     desc: state.sortOrder == 'desc',
    //     deleted: false,
    //     include_disabled: true
    // });
    //
    // const servers = useMemo(() => {
    //     return [...newServers, ...data];
    // }, [data, newServers]);
    //
    // const onCreate = useCallback(async () => {
    //     try {
    //         const newServer = await NiceModal.show<Server>(ModalServerEditor, {});
    //         setNewServers((prevState) => {
    //             return [newServer, ...prevState];
    //         });
    //
    //         sendFlash('success', 'Server created successfully');
    //     } catch (e) {
    //         sendFlash('error', `Failed to create new server: ${e}`);
    //     }
    // }, [sendFlash]);
    //
    // const onEdit = useCallback(
    //     async (server: Server) => {
    //         try {
    //             const newServer = await NiceModal.show<Server>(ModalServerEditor, { server });
    //             setNewServers((prevState) => {
    //                 return [newServer, ...prevState];
    //             });
    //
    //             sendFlash('success', 'Server edited successfully');
    //         } catch (e) {
    //             sendFlash('error', `Failed to edit server: ${e}`);
    //         }
    //     },
    //     [sendFlash]
    // );

    return (
        <Grid container spacing={2}>
            <Grid xs={12}>
                <Stack spacing={2}>
                    <ContainerWithHeaderAndButtons
                        title={'Servers'}
                        iconLeft={<StorageIcon />}
                        buttons={[
                            <ButtonGroup key={`server-header-buttons`}>
                                <Button
                                    variant={'contained'}
                                    color={'success'}
                                    startIcon={<AddIcon />}
                                    sx={{ marginRight: 2 }}
                                    // onClick={onCreate}
                                >
                                    Create Server
                                </Button>
                            </ButtonGroup>
                        ]}
                    >
                        <AdminServersTable servers={servers ?? { data: [], count: 0 }} isLoading={isLoading} />
                        <Paginator data={servers} page={page ?? 0} rows={rows ?? defaultRows} path={'/admin/servers'} />
                    </ContainerWithHeaderAndButtons>
                </Stack>
            </Grid>
        </Grid>
    );
}
const columnHelper = createColumnHelper<Server>();

const AdminServersTable = ({ servers, isLoading }: { servers: LazyResult<Server>; isLoading: boolean }) => {
    const columns = [
        columnHelper.accessor('server_id', {
            header: () => <TableHeadingCell name={'ID'} />,
            cell: (info) => <Typography>{info.getValue()}</Typography>
        }),
        columnHelper.accessor('short_name', {
            header: () => <TableHeadingCell name={'Name'} />,
            cell: (info) => <Typography>{info.getValue()}</Typography>
        }),
        columnHelper.accessor('name', {
            header: () => <TableHeadingCell name={'Name Long'} />,
            cell: (info) => <Typography>{info.getValue()}</Typography>
        }),
        columnHelper.accessor('address', {
            header: () => <TableHeadingCell name={'Address'} />,
            cell: (info) => <Typography>{info.getValue()}</Typography>
        }),
        columnHelper.accessor('port', {
            header: () => <TableHeadingCell name={'Reason'} />,
            cell: (info) => <Typography>{info.getValue()}</Typography>
        }),
        columnHelper.accessor('rcon', {
            header: () => <TableHeadingCell name={'RCON'} />,
            cell: (info) => <Typography>{info.getValue()}</Typography>
        }),
        columnHelper.accessor('region', {
            header: () => <TableHeadingCell name={'Region'} />,
            cell: (info) => <Typography>{info.getValue()}</Typography>
        }),
        columnHelper.accessor('cc', {
            header: () => <TableHeadingCell name={'CC'} />,
            cell: (info) => <Typography>{info.getValue()}</Typography>
        }),
        columnHelper.accessor('enable_stats', {
            header: () => <TableHeadingCell name={'Stats'} />,
            cell: (info) => <TableCellBool enabled={info.getValue()} />
        }),
        columnHelper.accessor('is_enabled', {
            header: () => <TableHeadingCell name={'En.'} />,
            cell: (info) => <TableCellBool enabled={info.getValue()} />
        })
    ];

    const table = useReactTable({
        data: servers.data,
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        manualPagination: true,
        autoResetPageIndex: true
    });

    return <DataTable table={table} isLoading={isLoading} />;
};
