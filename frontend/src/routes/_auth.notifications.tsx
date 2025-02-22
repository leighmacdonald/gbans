import { useMemo, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import ClearAllIcon from '@mui/icons-material/ClearAll';
import DoneIcon from '@mui/icons-material/Done';
import DoneAllIcon from '@mui/icons-material/DoneAll';
import EmailIcon from '@mui/icons-material/Email';
import LinkIcon from '@mui/icons-material/Link';
import MarkChatReadIcon from '@mui/icons-material/MarkChatRead';
import RemoveIcon from '@mui/icons-material/Remove';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import Typography from '@mui/material/Typography';
import Grid2 from '@mui/material/Unstable_Grid2';
import { useTheme } from '@mui/material/styles';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { Link, createFileRoute } from '@tanstack/react-router';
import {
    ColumnDef,
    getCoreRowModel,
    getPaginationRowModel,
    OnChangeFn,
    PaginationState,
    RowSelectionState,
    useReactTable
} from '@tanstack/react-table';
import {
    apiGetNotifications,
    apiNotificationsDelete,
    apiNotificationsDeleteAll,
    apiNotificationsMarkAllRead,
    apiNotificationsMarkRead,
    NotificationSeverity,
    UserNotification
} from '../api';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { DataTable } from '../component/DataTable.tsx';
import { IndeterminateCheckbox } from '../component/IndeterminateCheckbox.tsx';
import { PaginatorLocal } from '../component/PaginatorLocal.tsx';
import { PersonCell } from '../component/PersonCell.tsx';
import { TableCellBool } from '../component/TableCellBool.tsx';
import { TableCellRelativeDateField } from '../component/TableCellRelativeDateField.tsx';
import { TableCellString } from '../component/TableCellString.tsx';
import { Title } from '../component/Title.tsx';
import { ModalConfirm } from '../component/modal';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { RowsPerPage } from '../util/table.ts';

export const Route = createFileRoute('/_auth/notifications')({
    component: NotificationsPage
});

function NotificationsPage() {
    const queryClient = useQueryClient();
    const { sendError, sendFlash } = useUserFlashCtx();
    const [rowSelection, setRowSelection] = useState({});

    // const { page, rows, sortOrder, sortColumn } = Route.useSearch();
    const [pagination, setPagination] = useState({
        pageIndex: 0, //initial page index
        pageSize: RowsPerPage.TwentyFive //default page size
    });

    const { data: notifications, isLoading } = useQuery({
        queryKey: ['notifications'],
        queryFn: async () => {
            return await apiGetNotifications();
        }
    });

    const selectedToIds = () => {
        if (!notifications) {
            return [];
        }

        return Object.keys(rowSelection).map((s) => notifications[Number(s)].person_notification_id);
    };

    const onMarkAllRead = useMutation({
        mutationKey: ['notifications'],
        mutationFn: async () => {
            await apiNotificationsMarkAllRead();
        },
        onSuccess: () => {
            queryClient.setQueryData(['notifications'], (prev: UserNotification[]) => {
                return prev?.map((n) => {
                    return { ...n, read: true };
                });
            });
            sendFlash('success', `Successfully marked ${notifications?.length} as read`);
            setRowSelection({});
        },
        onError: sendError
    });

    const onMarkSelected = useMutation({
        mutationKey: ['notifications'],
        mutationFn: async (selectedIds: number[]) => {
            await apiNotificationsMarkRead(selectedIds);
        },
        onSuccess: (_, ids) => {
            queryClient.setQueryData(['notifications'], (prev: UserNotification[]) => {
                return prev?.map((n) => {
                    return ids.includes(n.person_notification_id) ? { ...n, read: true } : n;
                });
            });
            sendFlash('success', `Successfully marked ${ids?.length} as read`);
            setRowSelection({});
        },
        onError: sendError
    });

    const onDeleteAll = useMutation({
        mutationKey: ['notifications'],
        mutationFn: async () => {
            await apiNotificationsDeleteAll();
        },
        onSuccess: () => {
            queryClient.setQueryData(['notifications'], []);
            sendFlash('success', `Successfully deleted ${notifications?.length} messages`);
            setRowSelection({});
        },
        onError: sendError
    });

    const onDeleteSelected = useMutation({
        mutationKey: ['notifications'],
        mutationFn: async (selectedIds: number[]) => {
            await apiNotificationsDelete(selectedIds);
        },
        onSuccess: (_, ids) => {
            queryClient.setQueryData(['notifications'], (prev: UserNotification[]) => {
                return prev?.filter((n) => {
                    return !ids.includes(n.person_notification_id);
                });
            });
            sendFlash('success', `Successfully deleted ${ids?.length} messages`);
            setRowSelection({});
        },
        onError: sendError
    });

    const onConfirmDeleteSelected = async () => {
        const ids = selectedToIds();
        if (ids?.length == 0) {
            return;
        }
        const confirmed = await NiceModal.show<boolean>(ModalConfirm, {
            title: `Delete ${ids.length} notifications?`,
            children: 'This cannot be undone'
        });
        if (!confirmed) {
            return;
        }
        onDeleteSelected.mutate(ids);
    };

    const onConfirmDeleteAll = async () => {
        if (!notifications) {
            return;
        }
        const confirmed = await NiceModal.show<boolean>(ModalConfirm, {
            title: `Delete all ${notifications.length} notifications?`,
            children: 'This cannot be undone'
        });
        if (!confirmed) {
            return;
        }
        onDeleteAll.mutate();
    };
    const newMessages = useMemo(() => {
        return notifications?.filter((n) => !n.read).length;
    }, [notifications]);

    return (
        <>
            <Title>{`Notifications (${newMessages})`}</Title>
            <Grid2 container spacing={2}>
                <Grid2 xs={12}>
                    <ContainerWithHeaderAndButtons
                        iconLeft={<EmailIcon />}
                        title={`Notifications  ${Object.values(rowSelection).length ? `(Selected: ${Object.values(rowSelection).length})` : ''}`}
                        buttons={[
                            <ButtonGroup variant="contained" key={'hdr-buttons'}>
                                <Button
                                    startIcon={<DoneIcon />}
                                    color={'success'}
                                    key={'mark-selected'}
                                    onClick={() => {
                                        const ids = selectedToIds();
                                        if (ids?.length == 0) {
                                            return;
                                        }
                                        onMarkSelected.mutate(ids);
                                    }}
                                    disabled={Object.values(rowSelection).length == 0}
                                >
                                    Mark Selected Read
                                </Button>
                                <Button
                                    startIcon={<DoneAllIcon />}
                                    color={'success'}
                                    key={'mark-all'}
                                    onClick={() => onMarkAllRead.mutate()}
                                    disabled={(notifications ?? [])?.length == 0}
                                >
                                    Mark All Read
                                </Button>
                                <Button
                                    startIcon={<RemoveIcon />}
                                    color={'error'}
                                    key={'delete-selected'}
                                    onClick={onConfirmDeleteSelected}
                                    disabled={Object.values(rowSelection).length == 0}
                                >
                                    Delete Selected
                                </Button>
                                <Button
                                    startIcon={<ClearAllIcon />}
                                    color={'error'}
                                    key={'delete-all'}
                                    onClick={onConfirmDeleteAll}
                                    disabled={(notifications ?? [])?.length == 0}
                                >
                                    Delete All
                                </Button>
                            </ButtonGroup>
                        ]}
                    >
                        <NotificationsTable
                            notifications={notifications ?? []}
                            isLoading={isLoading}
                            rowSelection={rowSelection}
                            setRowSelection={setRowSelection}
                            pagination={pagination}
                            setPagination={setPagination}
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
                            count={notifications?.length ?? 0}
                            rows={pagination.pageSize}
                            page={pagination.pageIndex}
                        />
                    </ContainerWithHeaderAndButtons>
                </Grid2>
            </Grid2>
        </>
    );
}

const TableCellSeverity = ({ severity }: { severity: NotificationSeverity }) => {
    const theme = useTheme();

    switch (severity) {
        case NotificationSeverity.SeverityError:
            return <Typography style={{ color: theme.palette.error.main }}>ERROR</Typography>;
        case NotificationSeverity.SeverityWarn:
            return <Typography style={{ color: theme.palette.warning.main }}>WARN</Typography>;
        default:
            return <Typography style={{ color: theme.palette.info.main }}>INFO</Typography>;
    }
};

const NotificationsTable = ({
    notifications,
    isLoading,
    rowSelection,
    setRowSelection,
    pagination,
    setPagination
}: {
    notifications: UserNotification[];
    isLoading: boolean;
    rowSelection: RowSelectionState;
    setRowSelection: OnChangeFn<RowSelectionState>;
    pagination: PaginationState;
    setPagination: OnChangeFn<PaginationState>;
}) => {
    // const columnHelper = createColumnHelper<Filter>();
    const columns = useMemo<ColumnDef<UserNotification>[]>(
        () => [
            {
                id: 'select',
                header: ({ table }) => (
                    <IndeterminateCheckbox
                        {...{
                            checked: table.getIsAllRowsSelected(),
                            indeterminate: table.getIsSomeRowsSelected(),
                            onChange: table.getToggleAllRowsSelectedHandler()
                        }}
                    />
                ),
                cell: ({ row }) => (
                    <div className="px-1">
                        <IndeterminateCheckbox
                            {...{
                                checked: row.getIsSelected(),
                                disabled: !row.getCanSelect(),
                                indeterminate: row.getIsSomeSelected(),
                                onChange: row.getToggleSelectedHandler()
                            }}
                        />
                    </div>
                ),
                size: 30
            },
            {
                accessorKey: 'read',
                header: () => <MarkChatReadIcon />,
                cell: (info) => <TableCellBool enabled={info.getValue() as boolean} />,
                size: 30,
                enableResizing: false
            },
            {
                accessorKey: 'created_on',
                header: () => 'Created',
                cell: (info) => <TableCellRelativeDateField date={info.row.original.created_on} suffix={true} />,
                size: 125,
                enableResizing: false
            },
            {
                accessorKey: 'severity',
                header: () => 'level',
                cell: (info) => <TableCellSeverity severity={info.getValue() as NotificationSeverity} />,
                size: 55,
                enableResizing: false
            },
            {
                accessorKey: 'message',
                cell: (info) => <TableCellString>{info.getValue() as string}</TableCellString>
            },
            {
                accessorKey: 'author',
                cell: (info) =>
                    info.row.original.author != null ? (
                        <PersonCell
                            steam_id={info.row.original.author.steam_id}
                            personaname={info.row.original.author?.name}
                            avatar_hash={info.row.original.author?.avatarhash}
                        />
                    ) : (
                        ''
                    ),
                header: () => 'Author'
            },
            {
                accessorKey: 'link',
                size: 20,
                header: '',
                cell: (info) => {
                    return info.getValue() ? (
                        <Link to={info.getValue() as string}>
                            <LinkIcon color={'primary'} />
                        </Link>
                    ) : (
                        ''
                    );
                }
            }
        ],
        []
    );

    const table = useReactTable({
        data: notifications,
        columns: columns,
        getCoreRowModel: getCoreRowModel(),
        defaultColumn: {
            minSize: 0,
            size: Number.MAX_SAFE_INTEGER,
            maxSize: Number.MAX_SAFE_INTEGER
        },
        manualPagination: false,
        autoResetPageIndex: true,
        enableRowSelection: true,
        onRowSelectionChange: setRowSelection,
        onPaginationChange: setPagination,
        getPaginationRowModel: getPaginationRowModel(),
        state: {
            rowSelection,
            pagination
        }
    });

    return <DataTable table={table} isLoading={isLoading} />;
};
