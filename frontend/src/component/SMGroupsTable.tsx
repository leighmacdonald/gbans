import { useMemo, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import AssuredWorkloadIcon from '@mui/icons-material/AssuredWorkload';
import DeleteIcon from '@mui/icons-material/Delete';
import EditIcon from '@mui/icons-material/Edit';
import GroupAddIcon from '@mui/icons-material/GroupAdd';
import GroupsIcon from '@mui/icons-material/Groups';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import IconButton from '@mui/material/IconButton';
import Tooltip from '@mui/material/Tooltip';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { createColumnHelper } from '@tanstack/react-table';
import { apiDeleteSMGroup, SMAdmin, SMGroups } from '../api';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { initPagination, RowsPerPage } from '../util/table.ts';
import { renderDateTime } from '../util/text.tsx';
import { ContainerWithHeaderAndButtons } from './ContainerWithHeaderAndButtons.tsx';
import { FullTable } from './FullTable.tsx';
import { TableCellString } from './TableCellString.tsx';
import { TableHeadingCell } from './TableHeadingCell.tsx';
import { ModalConfirm, ModalSMGroupEditor, ModalSMGroupOverrides } from './modal';

export const SMGroupsTable = ({ groups, isLoading }: { groups: SMGroups[]; isLoading: boolean }) => {
    const { sendFlash } = useUserFlashCtx();
    const queryClient = useQueryClient();
    const [pagination, setPagination] = useState(initPagination(0, RowsPerPage.Ten));

    const onCreateGroup = async () => {
        try {
            const group = await NiceModal.show<SMGroups>(ModalSMGroupEditor);
            queryClient.setQueryData(['serverGroups'], [...(groups ?? []), group]);
            sendFlash('success', `Group created successfully: ${group.name}`);
        } catch (e) {
            sendFlash('error', 'Error trying to add group');
        }
    };

    const deleteGroupMutation = useMutation({
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
                deleteGroupMutation.mutate(group);
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
    }, [deleteGroupMutation, groups, queryClient, sendFlash]);

    return (
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
                data={groups ?? []}
                isLoading={isLoading}
                columns={groupColumns}
                pagination={pagination}
                setPagination={setPagination}
            />
        </ContainerWithHeaderAndButtons>
    );
};

const groupColumnHelper = createColumnHelper<SMGroups>();

const makeGroupColumns = (
    onEditGroup: (group: SMGroups) => Promise<void>,
    onDeleteGroup: (group: SMGroups) => Promise<void>,
    onOverride: (group: SMGroups) => Promise<void>
) => [
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
