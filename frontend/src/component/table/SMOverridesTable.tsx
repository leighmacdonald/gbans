import { useMemo, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import AssuredWorkloadIcon from '@mui/icons-material/AssuredWorkload';
import DeleteIcon from '@mui/icons-material/Delete';
import EditIcon from '@mui/icons-material/Edit';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import IconButton from '@mui/material/IconButton';
import Tooltip from '@mui/material/Tooltip';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { createColumnHelper } from '@tanstack/react-table';
import { apiDeleteSMOverride } from '../../api';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx';
import { Route } from '../../routes/_admin.admin.game-admins.tsx';
import { SMOverrides } from '../../schema/sourcemod.ts';
import { logErr } from '../../util/errors';
import { initPagination, RowsPerPage } from '../../util/table';
import { renderDateTime } from '../../util/time';
import { ContainerWithHeaderAndButtons } from '../ContainerWithHeaderAndButtons';
import { ModalConfirm, ModalSMOverridesEditor } from '../modal';
import { FullTable } from './FullTable';
import { TableCellString } from './TableCellString';

export const SMOverridesTable = ({ overrides, isLoading }: { overrides: SMOverrides[]; isLoading: boolean }) => {
    const { sendFlash, sendError } = useUserFlashCtx();
    const queryClient = useQueryClient();
    const [pagination, setPagination] = useState(initPagination(0, RowsPerPage.Ten));

    const onCreateOverride = async () => {
        try {
            const override = await NiceModal.show<SMOverrides>(ModalSMOverridesEditor, {});
            queryClient.setQueryData(['serverOverrides'], [...(overrides ?? []), override]);
            sendFlash('success', `Group created successfully: ${override.name}`);
        } catch (e) {
            logErr(e);
            sendFlash('error', 'Error trying to add group');
        }
    };

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
        onError: sendError
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
                logErr(e);
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
        return makeOverridesColumns(onEdit, onDelete);
    }, [delOverrideMutation, overrides, queryClient, sendFlash]);

    return (
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
            <FullTable
                data={overrides ?? []}
                isLoading={isLoading}
                columns={overridesColumns}
                pagination={pagination}
                setPagination={setPagination}
                toOptions={{ from: Route.fullPath }}
            />
        </ContainerWithHeaderAndButtons>
    );
};

const overrideColumnHelper = createColumnHelper<SMOverrides>();

const makeOverridesColumns = (
    onEdit: (override: SMOverrides) => Promise<void>,
    onDelete: (override: SMOverrides) => Promise<void>
) => [
    overrideColumnHelper.accessor('name', {
        header: 'Name',
        size: 500,
        cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    }),
    overrideColumnHelper.accessor('type', {
        header: 'Type',
        size: 75,
        cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    }),
    overrideColumnHelper.accessor('flags', {
        header: 'Flags',
        size: 75,
        cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    }),
    overrideColumnHelper.accessor('created_on', {
        header: 'Created On',
        size: 140,
        cell: (info) => <TableCellString>{renderDateTime(info.getValue())}</TableCellString>
    }),
    overrideColumnHelper.accessor('updated_on', {
        header: 'Updated On',
        size: 140,
        cell: (info) => <TableCellString>{renderDateTime(info.getValue())}</TableCellString>
    }),
    overrideColumnHelper.display({
        id: 'edit',
        maxSize: 30,
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
        maxSize: 30,
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
