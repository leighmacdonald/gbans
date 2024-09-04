import { useMemo, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import DeleteIcon from '@mui/icons-material/Delete';
import EditIcon from '@mui/icons-material/Edit';
import GroupAddIcon from '@mui/icons-material/GroupAdd';
import GroupRemoveIcon from '@mui/icons-material/GroupRemove';
import PersonIcon from '@mui/icons-material/Person';
import PersonAddIcon from '@mui/icons-material/PersonAdd';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import IconButton from '@mui/material/IconButton';
import Tooltip from '@mui/material/Tooltip';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { createColumnHelper } from '@tanstack/react-table';
import { apiAddAdminToGroup, apiDelAdminFromGroup, apiDeleteSMAdmin, SMAdmin, SMGroups } from '../api';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { initPagination, RowsPerPage } from '../util/table.ts';
import { renderDateTime } from '../util/text.tsx';
import { ContainerWithHeaderAndButtons } from './ContainerWithHeaderAndButtons.tsx';
import { FullTable } from './FullTable.tsx';
import { TableCellString } from './TableCellString.tsx';
import { ModalConfirm, ModalSMAdminEditor, ModalSMGroupSelect } from './modal';

export const SMAdminsTable = ({
    admins,
    groups,
    isLoading
}: {
    admins: SMAdmin[];
    groups: SMGroups[];
    isLoading: boolean;
}) => {
    const { sendFlash } = useUserFlashCtx();
    const queryClient = useQueryClient();
    const [pagination, setPagination] = useState(initPagination(0, RowsPerPage.Ten));

    const onCreateAdmin = async () => {
        try {
            const admin = await NiceModal.show<SMAdmin>(ModalSMAdminEditor, { groups });
            queryClient.setQueryData(['serverAdmins'], [...(admins ?? []), admin]);
            sendFlash('success', `Admin created successfully: ${admin.name}`);
        } catch (e) {
            sendFlash('error', 'Error trying to add admin');
        }
    };

    const deleteAdmin = useMutation({
        mutationKey: ['SMAdminDelete'],
        mutationFn: async (admin: SMAdmin) => {
            await apiDeleteSMAdmin(admin.admin_id);
            return admin;
        },
        onSuccess: (admin) => {
            queryClient.setQueryData(
                ['serverAdmins'],
                (admins ?? []).filter((a) => a.admin_id != admin.admin_id)
            );
            sendFlash('success', 'Admin deleted successfully');
        },
        onError: (error) => {
            sendFlash('error', `Error trying to delete admin: ${error}`);
        }
    });

    const addGroupMutation = useMutation({
        mutationKey: ['addAdminGroup'],
        mutationFn: async ({ admin, group }: { admin: SMAdmin; group: SMGroups }) => {
            return await apiAddAdminToGroup(admin.admin_id, group.group_id);
        },
        onSuccess: (edited) => {
            queryClient.setQueryData(
                ['serverAdmins'],
                (admins ?? []).map((a) => {
                    return a.admin_id == edited.admin_id ? edited : a;
                })
            );
            sendFlash('success', `Admin updated successfully: ${edited.name}`);
        }
    });

    const delGroupMutation = useMutation({
        mutationKey: ['addAdminGroup'],
        mutationFn: async ({ admin, group }: { admin: SMAdmin; group: SMGroups }) => {
            return await apiDelAdminFromGroup(admin.admin_id, group.group_id);
        },
        onSuccess: (edited) => {
            // FIXME
            queryClient.setQueryData(
                ['serverAdmins'],
                (admins ?? []).filter((a) => {
                    return a.admin_id != edited.admin_id;
                })
            );
            sendFlash('success', `Admin updated successfully: ${edited.name}`);
        }
    });

    const adminColumns = useMemo(() => {
        const onEdit = async (admin: SMAdmin) => {
            try {
                const edited = await NiceModal.show<SMAdmin>(ModalSMAdminEditor, { admin, groups });
                queryClient.setQueryData(
                    ['serverAdmins'],
                    (admins ?? []).map((a) => {
                        return a.admin_id == edited.admin_id ? edited : a;
                    })
                );
                sendFlash('success', `Admin updated successfully: ${admin.name}`);
            } catch (e) {
                sendFlash('error', 'Error trying to update admin');
            }
        };
        const onDelete = async (admin: SMAdmin) => {
            try {
                const confirmed = await NiceModal.show<boolean>(ModalConfirm, {
                    title: 'Delete admin?',
                    children: 'This cannot be undone'
                });
                if (!confirmed) {
                    return;
                }
                deleteAdmin.mutate(admin);
            } catch (e) {
                sendFlash('error', `Failed to create confirmation modal: ${e}`);
            }
        };

        const onAddGroup = async (admin: SMAdmin) => {
            try {
                const existingGroupIds = admin.groups.map((g) => g.group_id);
                const group = await NiceModal.show<SMGroups>(ModalSMGroupSelect, {
                    groups: groups?.filter((g) => !existingGroupIds.includes(g.group_id))
                });
                addGroupMutation.mutate({ admin, group });
            } catch (e) {
                sendFlash('error', `Error trying to add group: ${e}`);
            }
        };

        const onDelGroup = async (admin: SMAdmin) => {
            try {
                const existingGroupIds = admin.groups.map((g) => g.group_id);
                const group = await NiceModal.show<SMGroups>(ModalSMGroupSelect, {
                    groups: groups?.filter((g) => existingGroupIds.includes(g.group_id))
                });
                delGroupMutation.mutate({ admin, group });
            } catch (e) {
                sendFlash('error', 'Error trying to add group');
            }
        };

        return makeAdminColumns(groups?.length ?? 0, onEdit, onDelete, onAddGroup, onDelGroup);
    }, [addGroupMutation, admins, delGroupMutation, deleteAdmin, groups, queryClient, sendFlash]);

    return (
        <ContainerWithHeaderAndButtons
            title={'Admins'}
            iconLeft={<PersonIcon />}
            buttons={[
                <ButtonGroup key={`server-header-buttons`}>
                    <Button
                        variant={'contained'}
                        color={'success'}
                        startIcon={<PersonAddIcon />}
                        onClick={onCreateAdmin}
                    >
                        Create Admin
                    </Button>
                </ButtonGroup>
            ]}
        >
            <FullTable
                data={admins ?? []}
                isLoading={isLoading}
                columns={adminColumns}
                pagination={pagination}
                setPagination={setPagination}
            />
        </ContainerWithHeaderAndButtons>
    );
};

const adminColumnHelper = createColumnHelper<SMAdmin>();

const makeAdminColumns = (
    groupCount: number,
    onEdit: (admin: SMAdmin) => Promise<void>,
    onDelete: (admin: SMAdmin) => Promise<void>,
    onAddGroup: (admin: SMAdmin) => Promise<void>,
    onDelGroup: (admin: SMAdmin) => Promise<void>
) => [
    adminColumnHelper.accessor('name', {
        header: 'Name',
        cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    }),
    adminColumnHelper.accessor('auth_type', {
        header: 'Auth Type',
        cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    }),
    adminColumnHelper.accessor('identity', {
        header: 'Identity',
        cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    }),
    adminColumnHelper.accessor('steam_id', {
        header: 'SteamID',
        cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    }),
    adminColumnHelper.accessor('password', {
        header: 'Password',
        cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    }),
    adminColumnHelper.accessor('flags', {
        header: 'Flags',
        size: 75,
        cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    }),
    adminColumnHelper.accessor('immunity', {
        header: 'Immunity',
        size: 75,
        cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    }),
    adminColumnHelper.accessor('created_on', {
        header: 'Created On',
        size: 180,
        cell: (info) => <TableCellString>{renderDateTime(info.getValue())}</TableCellString>
    }),
    adminColumnHelper.accessor('updated_on', {
        header: 'Updated On',
        size: 180,
        cell: (info) => <TableCellString>{renderDateTime(info.getValue())}</TableCellString>
    }),
    adminColumnHelper.display({
        id: 'add_group',
        size: 30,
        cell: (info) => (
            <Tooltip title={'Add user to group'}>
                <span>
                    <IconButton
                        disabled={info.row.original.groups.length == groupCount}
                        color={'success'}
                        onClick={async () => {
                            await onAddGroup(info.row.original);
                        }}
                    >
                        <GroupAddIcon />
                    </IconButton>
                </span>
            </Tooltip>
        )
    }),
    adminColumnHelper.display({
        id: 'del_group',
        size: 30,
        cell: (info) => (
            <Tooltip title={'Remove user from group'}>
                <span>
                    <IconButton
                        disabled={info.row.original.groups.length == 0}
                        color={'error'}
                        onClick={async () => {
                            await onDelGroup(info.row.original);
                        }}
                    >
                        <GroupRemoveIcon />
                    </IconButton>
                </span>
            </Tooltip>
        )
    }),
    adminColumnHelper.display({
        id: 'edit',
        size: 30,
        cell: (info) => (
            <Tooltip title={'Edit admin'}>
                <IconButton
                    color={'warning'}
                    onClick={async () => {
                        await onEdit(info.row.original);
                    }}
                >
                    <EditIcon />
                </IconButton>
            </Tooltip>
        )
    }),
    adminColumnHelper.display({
        id: 'delete',
        size: 30,
        cell: (info) => (
            <Tooltip title={'Remove admin'}>
                <IconButton
                    color={'error'}
                    onClick={async () => {
                        await onDelete(info.row.original);
                    }}
                >
                    <DeleteIcon />
                </IconButton>
            </Tooltip>
        )
    })
];
