import { useMemo, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import AddIcon from '@mui/icons-material/Add';
import EditIcon from '@mui/icons-material/Edit';
import StorageIcon from '@mui/icons-material/Storage';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Grid from '@mui/material/Unstable_Grid2';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';
import {
    ColumnDef,
    getCoreRowModel,
    getPaginationRowModel,
    OnChangeFn,
    PaginationState,
    useReactTable
} from '@tanstack/react-table';
import { z } from 'zod';
import { apiGetServersAdmin, Server } from '../api';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { DataTable } from '../component/DataTable.tsx';
import { PaginatorLocal } from '../component/PaginatorLocal.tsx';
import { TableCellBool } from '../component/TableCellBool.tsx';
import { TableCellString } from '../component/TableCellString.tsx';
import { TableHeadingCell } from '../component/TableHeadingCell.tsx';
import { TableCellStringHidden } from '../component/field/TableCellStringHidden.tsx';
import { ModalServerEditor } from '../component/modal';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { commonTableSearchSchema, RowsPerPage } from '../util/table.ts';
import { renderDateTime } from '../util/text.tsx';

const serversSearchSchema = z.object({
    ...commonTableSearchSchema,
    sortColumn: z
        .enum(['server_id', 'short_name', 'name', 'address', 'port', 'region', 'cc', 'enable_stats', 'is_enabled'])
        .optional()
});

export const Route = createFileRoute('/_admin/admin/servers')({
    component: AdminServers,
    validateSearch: (search) => serversSearchSchema.parse(search)
});

function AdminServers() {
    const { sendFlash } = useUserFlashCtx();
    const queryClient = useQueryClient();
    //const { page, sortColumn, rows, sortOrder } = Route.useSearch();

    const [pagination, setPagination] = useState({
        pageIndex: 0, //initial page index
        pageSize: RowsPerPage.TwentyFive //default page size
    });

    const { data: servers, isLoading } = useQuery({
        queryKey: ['serversAdmin'],
        queryFn: async () => {
            return await apiGetServersAdmin();
        }
    });

    const onCreate = async () => {
        try {
            const newServer = await NiceModal.show<Server>(ModalServerEditor, {});
            queryClient.setQueryData(['serversAdmin'], [...(servers ?? []), newServer]);
            sendFlash('success', 'Server created successfully');
        } catch (e) {
            sendFlash('error', `Failed to create new server: ${e}`);
        }
    };

    const onEdit = async (server: Server) => {
        try {
            const editedServer = await NiceModal.show<Server>(ModalServerEditor, { server });
            console.log(editedServer);
            queryClient.setQueryData(
                ['serversAdmin'],
                (servers ?? []).map((s) => {
                    return s.server_id == editedServer.server_id ? editedServer : s;
                })
            );
            sendFlash('success', 'Server edited successfully');
        } catch (e) {
            sendFlash('error', `Failed to edit server: ${e}`);
        }
    };
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
                                    onClick={onCreate}
                                >
                                    Create Server
                                </Button>
                            </ButtonGroup>
                        ]}
                    >
                        <AdminServersTable
                            servers={servers ?? []}
                            isLoading={isLoading}
                            setPagination={setPagination}
                            pagination={pagination}
                            onEdit={onEdit}
                        />
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
                            count={servers?.length ?? 0}
                            rows={pagination.pageSize}
                            page={pagination.pageIndex}
                        />
                    </ContainerWithHeaderAndButtons>
                </Stack>
            </Grid>
        </Grid>
    );
}

const AdminServersTable = ({
    servers,
    isLoading,
    setPagination,
    pagination,
    onEdit
}: {
    servers: Server[];
    isLoading: boolean;
    onEdit: (server: Server) => Promise<void>;
    pagination: PaginationState;
    setPagination: OnChangeFn<PaginationState>;
}) => {
    const columns = useMemo<ColumnDef<Server>[]>(
        () => [
            {
                accessorKey: 'server_id',
                header: () => <TableHeadingCell name={'ID'} />,
                cell: (info) => <TableCellString>{info.getValue() as string}</TableCellString>
            },
            {
                accessorKey: 'short_name',
                header: () => <TableHeadingCell name={'Name'} tooltip={'Short unique server identifier'} />,
                cell: (info) => <TableCellString>{info.getValue() as string}</TableCellString>
            },
            {
                accessorKey: 'name',
                header: () => (
                    <TableHeadingCell name={'Name Long'} tooltip={'Full name of the server, AKA srcds hostname'} />
                ),
                cell: (info) => <TableCellString>{info.getValue() as string}</TableCellString>
            },
            {
                accessorKey: 'address',
                header: () => <TableHeadingCell name={'Address'} tooltip={'IP or DNS/Hostname of the server'} />,
                cell: (info) => <TableCellString>{info.getValue() as string}</TableCellString>
            },
            {
                accessorKey: 'port',
                header: () => <TableHeadingCell name={'Port'} />,
                cell: (info) => <TableCellString>{info.getValue() as string}</TableCellString>
            },
            {
                accessorKey: 'rcon',
                header: () => <TableHeadingCell name={'RCON'} tooltip={'Standard RCON password'} />,
                cell: (info) => <TableCellStringHidden>{info.getValue() as string}</TableCellStringHidden>
            },
            {
                accessorKey: 'password',
                header: () => (
                    <TableHeadingCell
                        name={'Auth PSK'}
                        tooltip={'A password that the server uses to authenticate with the central gbans server'}
                    />
                ),
                cell: (info) => <TableCellStringHidden>{info.getValue() as string}</TableCellStringHidden>
            },
            {
                accessorKey: 'region',
                header: () => <TableHeadingCell name={'Region'} />,
                cell: (info) => <TableCellString>{info.getValue() as string}</TableCellString>
            },
            {
                accessorKey: 'cc',
                header: () => <TableHeadingCell name={'CC'} tooltip={'2 character country code'} />,
                cell: (info) => <TableCellString>{info.getValue() as string}</TableCellString>
            },
            {
                accessorKey: 'latitude',
                header: () => <TableHeadingCell name={'Lat'} tooltip={'Latitude'} />,
                cell: (info) => <TableCellString>{Number(info.getValue()).toFixed(2)}</TableCellString>
            },
            {
                accessorKey: 'longitude',
                header: () => <TableHeadingCell name={'Lon'} tooltip={'Longitude'} />,
                cell: (info) => <TableCellString>{Number(info.getValue()).toFixed(2)}</TableCellString>
            },
            {
                accessorKey: 'token_created_on',
                header: () => (
                    <TableHeadingCell name={'Last Auth'} tooltip={'Last time the server authenticated itself'} />
                ),
                cell: (info) => <TableCellString>{renderDateTime(info.getValue() as Date)}</TableCellString>
            },
            {
                accessorKey: 'enable_stats',
                header: () => <TableHeadingCell name={'St'} tooltip={'Stat Tracking Enabled'} />,
                cell: (info) => <TableCellBool enabled={info.getValue() as boolean} />
            },
            {
                accessorKey: 'is_enabled',
                header: () => <TableHeadingCell name={'En.'} tooltip={'Enabled'} />,
                cell: (info) => <TableCellBool enabled={info.getValue() as boolean} />
            },
            {
                id: 'actions',
                cell: (info) => {
                    return (
                        <ButtonGroup fullWidth>
                            <IconButton
                                color={'warning'}
                                onClick={async () => {
                                    await onEdit(info.row.original);
                                }}
                            >
                                <Tooltip title={'Edit Ban'}>
                                    <EditIcon />
                                </Tooltip>
                            </IconButton>
                        </ButtonGroup>
                    );
                }
            }
        ],
        [onEdit]
    );

    const table = useReactTable({
        data: servers,
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        getPaginationRowModel: getPaginationRowModel(),
        onPaginationChange: setPagination, //update the pagination state when internal APIs mutate the pagination state
        state: {
            pagination
        }
    });

    return <DataTable table={table} isLoading={isLoading} />;
};
