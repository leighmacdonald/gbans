import { useMemo } from 'react';
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
import { apiDeleteSMOverride, SMOverrides } from '../api';
import { useUserFlashCtx } from '../hooks/useUserFlashCtx.ts';
import { renderDateTime } from '../util/text.tsx';
import { ContainerWithHeaderAndButtons } from './ContainerWithHeaderAndButtons.tsx';
import { FullTable } from './FullTable.tsx';
import { TableCellString } from './TableCellString.tsx';
import { TableHeadingCell } from './TableHeadingCell.tsx';
import { ModalConfirm, ModalSMOverridesEditor } from './modal';

export const SMOverridesTable = ({ overrides, isLoading }: { overrides: SMOverrides[]; isLoading: boolean }) => {
    const { sendFlash } = useUserFlashCtx();
    const queryClient = useQueryClient();

    const onCreateOverride = async () => {
        try {
            const override = await NiceModal.show<SMOverrides>(ModalSMOverridesEditor, {});
            queryClient.setQueryData(['serverOverrides'], [...(overrides ?? []), override]);
            sendFlash('success', `Group created successfully: ${override.name}`);
        } catch (e) {
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
            <FullTable data={overrides ?? []} isLoading={isLoading} columns={overridesColumns} />
        </ContainerWithHeaderAndButtons>
    );
};

const overrideColumnHelper = createColumnHelper<SMOverrides>();

const makeOverridesColumns = (
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
