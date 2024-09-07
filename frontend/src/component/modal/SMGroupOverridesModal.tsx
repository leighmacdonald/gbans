import { useMemo, useState } from 'react';
import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import AddIcon from '@mui/icons-material/Add';
import CloseIcon from '@mui/icons-material/Close';
import DeleteIcon from '@mui/icons-material/Delete';
import EditIcon from '@mui/icons-material/Edit';
import GroupsIcon from '@mui/icons-material/Groups';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import IconButton from '@mui/material/IconButton';
import Grid from '@mui/material/Unstable_Grid2';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { createColumnHelper } from '@tanstack/react-table';
import 'video-react/dist/video-react.css';
import { apiDeleteSMGroupOverride, apiGetSMGroupOverrides, SMGroupOverrides, SMGroups } from '../../api';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { Route } from '../../routes/_admin.admin.game-admins.tsx';
import { logErr } from '../../util/errors.ts';
import { initPagination, RowsPerPage } from '../../util/table.ts';
import { renderDateTime } from '../../util/text.tsx';
import { FullTable } from '../FullTable.tsx';
import { Heading } from '../Heading';
import { TableCellString } from '../TableCellString.tsx';
import { ModalConfirm, ModalSMGroupOverridesEditor } from './index.ts';

const overrideColumnHelper = createColumnHelper<SMGroupOverrides>();

const makeColumns = (
    onEdit: (override: SMGroupOverrides) => Promise<void>,
    onDelete: (override: SMGroupOverrides) => Promise<void>
) => [
    overrideColumnHelper.accessor('name', {
        header: 'Name',
        size: 100,
        cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    }),
    overrideColumnHelper.accessor('type', {
        header: 'Type',
        cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    }),
    overrideColumnHelper.accessor('access', {
        header: 'Access',
        cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    }),
    overrideColumnHelper.accessor('created_on', {
        header: 'Created',
        size: 120,
        cell: (info) => <TableCellString>{renderDateTime(info.getValue())}</TableCellString>
    }),
    overrideColumnHelper.accessor('updated_on', {
        header: 'Updated',
        size: 120,
        cell: (info) => <TableCellString>{renderDateTime(info.getValue())}</TableCellString>
    }),
    overrideColumnHelper.display({
        id: 'edit',
        size: 30,
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
    overrideColumnHelper.display({
        id: 'delete',
        size: 30,
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

export const SMGroupOverridesModal = NiceModal.create(({ group }: { group: SMGroups }) => {
    const modal = useModal();
    const queryClient = useQueryClient();
    const { sendFlash } = useUserFlashCtx();
    const [pagination, setPagination] = useState(initPagination(0, RowsPerPage.Ten));

    const { data: overrides, isLoading } = useQuery({
        queryKey: ['serverGroupOverrides', { group_id: group.group_id }],
        queryFn: async () => {
            return await apiGetSMGroupOverrides(group.group_id);
        }
    });

    const onCreate = async () => {
        try {
            const created = await NiceModal.show<SMGroupOverrides>(ModalSMGroupOverridesEditor, { group });
            queryClient.setQueryData(
                ['serverGroupOverrides', { group_id: group.group_id }],
                [...(overrides ?? []), created]
            );
            sendFlash('success', `Group override created successfully: ${created.name}`);
        } catch (e) {
            logErr(e);
            sendFlash('error', 'Error trying to add group override');
        }
    };
    const delOverrideMutation = useMutation({
        mutationKey: ['deleteGroupOverride'],
        mutationFn: async ({ groupOverride }: { groupOverride: SMGroupOverrides }) => {
            await apiDeleteSMGroupOverride(groupOverride.group_override_id);
            return groupOverride;
        },
        onSuccess: (edited) => {
            queryClient.setQueryData(
                ['serverGroupOverrides', { group_id: edited.group_id }],
                (overrides ?? []).filter((o) => {
                    return o.group_override_id != edited.group_override_id;
                })
            );
            sendFlash('success', `Group override deleted successfully: ${edited.name}`);
        },
        onError: (error) => {
            sendFlash('error', `Failed to delete group override: ${error}`);
        }
    });
    const columns = useMemo(() => {
        const onEdit = async (override: SMGroupOverrides) => {
            try {
                const edited = await NiceModal.show<SMGroupOverrides>(ModalSMGroupOverridesEditor, { override });
                queryClient.setQueryData(
                    ['serverGroupOverrides', { group_id: group.group_id }],
                    (overrides ?? []).map((o) => {
                        return o.group_override_id == edited.group_override_id ? edited : o;
                    })
                );
                sendFlash('success', `Group override updated successfully: ${override.name}`);
            } catch (e) {
                logErr(e);
                sendFlash('error', 'Error trying to edit group override');
            }
        };

        const onDelete = async (groupOverride: SMGroupOverrides) => {
            try {
                const confirmed = await NiceModal.show<boolean>(ModalConfirm, {
                    title: 'Delete override?',
                    children: 'This cannot be undone'
                });
                if (!confirmed) {
                    return;
                }
                delOverrideMutation.mutate({ groupOverride });
            } catch (e) {
                sendFlash('error', `Failed to create confirmation modal: ${e}`);
            }
        };
        return makeColumns(onEdit, onDelete);
    }, [delOverrideMutation, group.group_id, overrides, queryClient, sendFlash]);

    return (
        <Dialog fullWidth {...muiDialogV5(modal)}>
            <DialogTitle component={Heading} iconLeft={<GroupsIcon />}>
                Group Overrides
            </DialogTitle>

            <DialogContent>
                <FullTable
                    data={overrides ?? []}
                    isLoading={isLoading}
                    columns={columns}
                    pagination={pagination}
                    setPagination={setPagination}
                    toOptions={{ from: Route.fullPath }}
                />
            </DialogContent>

            <DialogActions>
                <Grid container>
                    <Grid xs={12} mdOffset="auto">
                        <ButtonGroup variant={'contained'}>
                            <Button startIcon={<AddIcon />} color={'success'} onClick={onCreate}>
                                New
                            </Button>
                            <Button key={'close-button'} onClick={modal.hide} color={'error'} startIcon={<CloseIcon />}>
                                Close
                            </Button>
                        </ButtonGroup>
                    </Grid>
                </Grid>
            </DialogActions>
        </Dialog>
    );
});
