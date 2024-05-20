import { useMemo } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import AssuredWorkloadIcon from '@mui/icons-material/AssuredWorkload';
import DeleteIcon from '@mui/icons-material/Delete';
import EditIcon from '@mui/icons-material/Edit';
import GroupAddIcon from '@mui/icons-material/GroupAdd';
import GroupRemoveIcon from '@mui/icons-material/GroupRemove';
import GroupsIcon from '@mui/icons-material/Groups';
import PersonIcon from '@mui/icons-material/Person';
import PersonAddIcon from '@mui/icons-material/PersonAdd';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';
import { createColumnHelper } from '@tanstack/react-table';
import {
    apiAddAdminToGroup,
    apiDelAdminFromGroup,
    apiDeleteSMAdmin,
    apiDeleteSMGroup,
    apiDeleteSMOverride,
    apiGetSMAdmins,
    apiGetSMGroups,
    apiGetSMOverrides,
    SMAdmin,
    SMGroups,
    SMOverrides
} from '../api';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { FullTable } from '../component/FullTable.tsx';
import { TableCellString } from '../component/TableCellString.tsx';
import { TableHeadingCell } from '../component/TableHeadingCell.tsx';
import { Title } from '../component/Title';
import {
    ModalConfirm,
    ModalSMAdminEditor,
    ModalSMGroupOverrides,
    ModalSMGroupEditor,
    ModalSMGroupSelect,
    ModalSMOverridesEditor
} from '../component/modal';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { RowsPerPage } from '../util/table.ts';
import { renderDateTime } from '../util/text.tsx';

export const Route = createFileRoute('/_admin/admin/game-admins')({
    component: AdminsEditor
});

function AdminsEditor() {
    const { sendFlash } = useUserFlashCtx();
    const queryClient = useQueryClient();

    const { data: groups, isLoading: isLoadingGroups } = useQuery({
        queryKey: ['serverGroups'],
        queryFn: async () => {
            return await apiGetSMGroups();
        }
    });

    const { data: admins, isLoading: isLoadingAdmins } = useQuery({
        queryKey: ['serverAdmins'],
        queryFn: async () => {
            return await apiGetSMAdmins();
        }
    });

    const { data: overrides, isLoading: isLoadingOverrides } = useQuery({
        queryKey: ['serverOverrides'],
        queryFn: async () => {
            const resp = await apiGetSMOverrides();
            console.log(resp);
            return resp;
        }
    });

    const onCreateGroup = async () => {
        try {
            const group = await NiceModal.show<SMGroups>(ModalSMGroupEditor);
            queryClient.setQueryData(['serverGroups'], [...(groups ?? []), group]);
            sendFlash('success', `Group created successfully: ${group.name}`);
        } catch (e) {
            sendFlash('error', 'Error trying to add group');
        }
    };

    const onCreateOverride = async () => {
        try {
            const override = await NiceModal.show<SMOverrides>(ModalSMOverridesEditor, {});
            queryClient.setQueryData(['serverOverrides'], [...(overrides ?? []), override]);
            sendFlash('success', `Group created successfully: ${override.name}`);
        } catch (e) {
            sendFlash('error', 'Error trying to add group');
        }
    };

    const deleteGroup = useMutation({
        mutationKey: ['SMGroupDelete'],
        mutationFn: async (group: SMGroups) => {
            await apiDeleteSMGroup(group.group_id);
            return group;
        },
        onSuccess: (group) => {
            queryClient.setQueryData(
                ['serverGroups'],
                (groups ?? []).filter((g) => g.group_id != group.group_id)
            );
            sendFlash('success', 'Group deleted successfully');
        },
        onError: (error) => {
            sendFlash('error', `Error trying to delete group: ${error}`);
        }
    });

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
                (admins ?? []).map((a) => {
                    return a.admin_id == edited.admin_id ? edited : a;
                })
            );
            sendFlash('success', `Admin updated successfully: ${edited.name}`);
        }
    });

    const groupColumns = useMemo(() => {
        const onOverride = async (group: SMGroups) => {
            await NiceModal.show<SMAdmin>(ModalSMGroupOverrides, { group });
        };

        const onDeleteGroup = async (group: SMGroups) => {
            try {
                const confirmed = await NiceModal.show<boolean>(ModalConfirm, {
                    title: 'Delete group?',
                    children: 'This cannot be undone'
                });
                if (!confirmed) {
                    return;
                }
                deleteGroup.mutate(group);
            } catch (e) {
                sendFlash('error', `Failed to create confirmation modal: ${e}`);
            }
        };
        const onEditGroup = async (group: SMGroups) => {
            try {
                const editedGroup = await NiceModal.show<SMGroups>(ModalSMGroupEditor, { group });
                queryClient.setQueryData(
                    ['serverGroups'],
                    (groups ?? []).map((g) => {
                        return g.group_id != editedGroup.group_id ? g : editedGroup;
                    })
                );
                sendFlash('success', `Group created successfully: ${group.name}`);
            } catch (e) {
                sendFlash('error', 'Error trying to add group');
            }
        };

        return makeGroupColumns(onEditGroup, onDeleteGroup, onOverride);
    }, [deleteGroup, groups, queryClient, sendFlash]);

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

    const delOverrideMutation = useMutation({
        mutationKey: ['delOverride'],
        mutationFn: async ({ override }: { override: SMOverrides }) => {
            await apiDeleteSMOverride(override.override_id);
            return override;
        },
        onSuccess: (deleted) => {
            queryClient.setQueryData(
                ['serverOverrides'],
                (overrides ?? []).filter((o) => {
                    return o.override_id != deleted.override_id;
                })
            );
            sendFlash('success', `Override deleted successfully: ${deleted.name}`);
        },
        onError: (error) => {
            sendFlash('error', `Failed to delete override: ${error}`);
        }
    });

    const overridesColumns = useMemo(() => {
        const onEdit = async (override: SMOverrides) => {
            try {
                const edited = await NiceModal.show<SMOverrides>(ModalSMOverridesEditor, { override });
                queryClient.setQueryData(
                    ['serverOverrides'],
                    (overrides ?? []).map((o) => {
                        return o.override_id == edited.override_id ? edited : o;
                    })
                );
                sendFlash('success', `Admin updated successfully: ${override.name}`);
            } catch (e) {
                sendFlash('error', 'Error trying to update admin');
            }
        };

        const onDelete = async (override: SMOverrides) => {
            try {
                const confirmed = await NiceModal.show<boolean>(ModalConfirm, {
                    title: 'Delete override?',
                    children: 'This cannot be undone'
                });
                if (!confirmed) {
                    return;
                }
                delOverrideMutation.mutate({ override });
            } catch (e) {
                sendFlash('error', `Failed to create confirmation modal: ${e}`);
            }
        };
        return makeOverridesColumn(onEdit, onDelete);
    }, [admins, delOverrideMutation, groups, queryClient, sendFlash]);

    return (
        <>
            <Title>Edit Server Admin Permissions</Title>
            <Stack spacing={2}>
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
                        pageSize={RowsPerPage.Ten}
                        data={admins ?? []}
                        isLoading={isLoadingAdmins}
                        columns={adminColumns}
                        initialSortColumn={'name'}
                        initialSortDesc={false}
                    />
                </ContainerWithHeaderAndButtons>
                <ContainerWithHeaderAndButtons
                    title={'Groups'}
                    iconLeft={<GroupsIcon />}
                    buttons={[
                        <ButtonGroup key={`group-header-buttons`} variant={'contained'}>
                            <Button color={'success'} startIcon={<GroupAddIcon />} onClick={onCreateGroup}>
                                Create Group
                            </Button>
                        </ButtonGroup>
                    ]}
                >
                    <FullTable
                        pageSize={RowsPerPage.Ten}
                        data={groups ?? []}
                        isLoading={isLoadingGroups}
                        columns={groupColumns}
                        initialSortColumn={'name'}
                        initialSortDesc={false}
                    />
                </ContainerWithHeaderAndButtons>
                <ContainerWithHeaderAndButtons
                    title={'Command Overrides'}
                    iconLeft={<AssuredWorkloadIcon />}
                    buttons={[
                        <ButtonGroup key={`override-header-buttons`} variant={'contained'}>
                            <Button color={'success'} startIcon={<AssuredWorkloadIcon />} onClick={onCreateOverride}>
                                Add Override
                            </Button>
                        </ButtonGroup>
                    ]}
                >
                    <FullTable data={overrides ?? []} isLoading={isLoadingOverrides} columns={overridesColumns} />
                </ContainerWithHeaderAndButtons>
            </Stack>
        </>
    );
}
const groupColumnHelper = createColumnHelper<SMGroups>();

const makeGroupColumns = (
    onEditGroup: (group: SMGroups) => Promise<void>,
    onDeleteGroup: (group: SMGroups) => Promise<void>,
    onOverride: (group: SMGroups) => Promise<void>
) => [
    // groupColumnHelper.accessor('group_id', {
    //     header: () => <TableHeadingCell name={'Group ID'} />,
    //     cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    // }),
    groupColumnHelper.accessor('name', {
        header: () => <TableHeadingCell name={'Name'} />,
        cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    }),
    groupColumnHelper.accessor('flags', {
        header: () => <TableHeadingCell name={'Flags'} />,
        cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    }),
    groupColumnHelper.accessor('immunity_level', {
        header: () => <TableHeadingCell name={'Immunity'} />,
        cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    }),
    groupColumnHelper.accessor('created_on', {
        header: () => <TableHeadingCell name={'Created On'} />,
        cell: (info) => <TableCellString>{renderDateTime(info.getValue())}</TableCellString>
    }),
    groupColumnHelper.accessor('updated_on', {
        header: () => <TableHeadingCell name={'Updated On'} />,
        cell: (info) => <TableCellString>{renderDateTime(info.getValue())}</TableCellString>
    }),
    groupColumnHelper.display({
        id: 'overrides',
        cell: (info) => (
            <Tooltip title={'Edit group overrides'}>
                <IconButton
                    color={'secondary'}
                    onClick={async () => {
                        await onOverride(info.row.original);
                    }}
                >
                    <AssuredWorkloadIcon />
                </IconButton>
            </Tooltip>
        )
    }),
    groupColumnHelper.display({
        id: 'edit',
        maxSize: 10,
        cell: (info) => (
            <IconButton
                color={'warning'}
                onClick={async () => {
                    await onEditGroup(info.row.original);
                }}
            >
                <EditIcon />
            </IconButton>
        )
    }),
    groupColumnHelper.display({
        id: 'delete',
        maxSize: 10,
        cell: (info) => (
            <IconButton
                color={'error'}
                onClick={async () => {
                    await onDeleteGroup(info.row.original);
                }}
            >
                <DeleteIcon />
            </IconButton>
        )
    })
];

const adminColumnHelper = createColumnHelper<SMAdmin>();

const makeAdminColumns = (
    groupCount: number,
    onEdit: (admin: SMAdmin) => Promise<void>,
    onDelete: (admin: SMAdmin) => Promise<void>,
    onAddGroup: (admin: SMAdmin) => Promise<void>,
    onDelGroup: (admin: SMAdmin) => Promise<void>
) => [
    // adminColumnHelper.accessor('admin_id', {
    //     header: () => <TableHeadingCell name={'ID'} />,
    //     cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    // }),
    adminColumnHelper.accessor('name', {
        header: () => <TableHeadingCell name={'Name'} />,
        cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    }),
    adminColumnHelper.accessor('auth_type', {
        header: () => <TableHeadingCell name={'Auth Type'} />,
        cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    }),
    adminColumnHelper.accessor('identity', {
        header: () => <TableHeadingCell name={'Identity'} />,
        cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    }),
    adminColumnHelper.accessor('steam_id', {
        header: () => <TableHeadingCell name={'SteamID'} />,
        cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    }),
    adminColumnHelper.accessor('password', {
        header: () => <TableHeadingCell name={'Password'} />,
        cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    }),
    adminColumnHelper.accessor('flags', {
        header: () => <TableHeadingCell name={'Flags'} />,
        cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    }),
    adminColumnHelper.accessor('immunity', {
        header: () => <TableHeadingCell name={'Immunity'} />,
        cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    }),
    adminColumnHelper.accessor('created_on', {
        header: () => <TableHeadingCell name={'Created On'} />,
        cell: (info) => <TableCellString>{renderDateTime(info.getValue())}</TableCellString>
    }),
    adminColumnHelper.accessor('updated_on', {
        header: () => <TableHeadingCell name={'Updated On'} />,
        cell: (info) => <TableCellString>{renderDateTime(info.getValue())}</TableCellString>
    }),
    adminColumnHelper.display({
        id: 'add_group',
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
        maxSize: 10,
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
        maxSize: 10,
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

const overrideColumnHelper = createColumnHelper<SMOverrides>();

const makeOverridesColumn = (
    onEdit: (override: SMOverrides) => Promise<void>,
    onDelete: (override: SMOverrides) => Promise<void>
) => [
    overrideColumnHelper.accessor('name', {
        header: () => <TableHeadingCell name={'Name'} />,
        cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    }),
    overrideColumnHelper.accessor('type', {
        header: () => <TableHeadingCell name={'Type'} />,
        cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    }),
    overrideColumnHelper.accessor('flags', {
        header: () => <TableHeadingCell name={'Flags'} />,
        cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    }),
    overrideColumnHelper.accessor('created_on', {
        header: () => <TableHeadingCell name={'Created On'} />,
        cell: (info) => <TableCellString>{renderDateTime(info.getValue())}</TableCellString>
    }),
    overrideColumnHelper.accessor('updated_on', {
        header: () => <TableHeadingCell name={'Updated On'} />,
        cell: (info) => <TableCellString>{renderDateTime(info.getValue())}</TableCellString>
    }),
    overrideColumnHelper.display({
        id: 'edit',
        maxSize: 10,
        cell: (info) => (
            <Tooltip title={'Edit Override'}>
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
    overrideColumnHelper.display({
        id: 'delete',
        maxSize: 10,
        cell: (info) => (
            <Tooltip title={'Delete override'}>
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
