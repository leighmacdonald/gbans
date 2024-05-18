import { useMemo } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import DeleteIcon from '@mui/icons-material/Delete';
import EditIcon from '@mui/icons-material/Edit';
import GroupAddIcon from '@mui/icons-material/GroupAdd';
import GroupsIcon from '@mui/icons-material/Groups';
import PersonAddIcon from '@mui/icons-material/PersonAdd';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { createFileRoute } from '@tanstack/react-router';
import { createColumnHelper } from '@tanstack/react-table';
import { apiDeleteSMGroup, apiGetSMAdmins, apiGetSMGroups, SMAdmin, SMGroups } from '../api';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons.tsx';
import { FullTable } from '../component/FullTable.tsx';
import { TableCellString } from '../component/TableCellString.tsx';
import { TableHeadingCell } from '../component/TableHeadingCell.tsx';
import { Title } from '../component/Title';
import { ModalConfirm, ModalSMAdminEditor, ModalSMGroupEditor } from '../component/modal';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { RowsPerPage } from '../util/table.ts';
import { renderDateTime } from '../util/text.tsx';

export const Route = createFileRoute('/_admin/admin/game-admins')({
    component: AdminsEditor
});

const groupColumnHelper = createColumnHelper<SMGroups>();

const makeGroupColumns = (
    onEditGroup: (group: SMGroups) => Promise<void>,
    onDeleteGroup: (group: SMGroups) => Promise<void>
) => [
    groupColumnHelper.accessor('group_id', {
        header: () => <TableHeadingCell name={'Group ID'} />,
        cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    }),
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

const makeAdminColumns = (onEdit: (admin: SMAdmin) => Promise<void>, onDelete: (admin: SMAdmin) => Promise<void>) => [
    adminColumnHelper.accessor('admin_id', {
        header: () => <TableHeadingCell name={'ID'} />,
        cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    }),
    adminColumnHelper.accessor('steam_id', {
        header: () => <TableHeadingCell name={'SteamID'} />,
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
    adminColumnHelper.accessor('password', {
        header: () => <TableHeadingCell name={'Password'} />,
        cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    }),
    adminColumnHelper.accessor('flags', {
        header: () => <TableHeadingCell name={'Flags'} />,
        cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    }),
    adminColumnHelper.accessor('name', {
        header: () => <TableHeadingCell name={'Name'} />,
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
        id: 'edit',
        maxSize: 10,
        cell: (info) => (
            <IconButton
                color={'warning'}
                onClick={async () => {
                    await onEdit(info.row.original);
                }}
            >
                <EditIcon />
            </IconButton>
        )
    }),
    adminColumnHelper.display({
        id: 'delete',
        maxSize: 10,
        cell: (info) => (
            <IconButton
                color={'error'}
                onClick={async () => {
                    await onDelete(info.row.original);
                }}
            >
                <DeleteIcon />
            </IconButton>
        )
    })
];

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

    const onCreateGroup = async () => {
        try {
            const group = await NiceModal.show<SMGroups>(ModalSMGroupEditor);
            queryClient.setQueryData(['serverGroups'], [...(groups ?? []), group]);
            sendFlash('success', `Group created successfully: ${group.name}`);
        } catch (e) {
            sendFlash('error', 'Error trying to add group');
        }
    };

    const onCreateAdmin = async () => {
        try {
            const admin = await NiceModal.show<SMAdmin>(ModalSMAdminEditor);
            queryClient.setQueryData(['serverAdmins'], [...(admins ?? []), admin]);
            sendFlash('success', `Admin created successfully: ${admin.name}`);
        } catch (e) {
            sendFlash('error', 'Error trying to add admin');
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

    const groupColumns = useMemo(() => {
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

        return makeGroupColumns(onEditGroup, onDeleteGroup);
    }, [deleteGroup, groups, queryClient, sendFlash]);

    const adminColumns = useMemo(() => {
        const onEdit = async (_: SMAdmin) => {};
        const onDelete = async (_: SMAdmin) => {};
        return makeAdminColumns(onEdit, onDelete);
    }, []);

    return (
        <>
            <Title>Edit Server Admin Permissions</Title>
            <Stack spacing={2}>
                <ContainerWithHeaderAndButtons
                    title={'Admins'}
                    iconLeft={<GroupsIcon />}
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
                    />
                </ContainerWithHeaderAndButtons>
                <ContainerWithHeaderAndButtons
                    title={'Groups'}
                    iconLeft={<GroupsIcon />}
                    buttons={[
                        <ButtonGroup key={`server-header-buttons`}>
                            <Button
                                variant={'contained'}
                                color={'success'}
                                startIcon={<GroupAddIcon />}
                                onClick={onCreateGroup}
                            >
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
                    />
                </ContainerWithHeaderAndButtons>
            </Stack>
        </>
    );
}
