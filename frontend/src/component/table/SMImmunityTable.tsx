import { useMemo, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import AssuredWorkloadIcon from '@mui/icons-material/AssuredWorkload';
import DeleteIcon from '@mui/icons-material/Delete';
import FlakyIcon from '@mui/icons-material/Flaky';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import IconButton from '@mui/material/IconButton';
import Tooltip from '@mui/material/Tooltip';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { createColumnHelper } from '@tanstack/react-table';
import { apiDeleteSMGroupImmunity } from '../../api';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx';
import { Route } from '../../routes/_admin.admin.game-admins.tsx';
import { SMGroupImmunity, SMGroups } from '../../schema/sourcemod.ts';
import { logErr } from '../../util/errors';
import { initPagination, RowsPerPage } from '../../util/table';
import { renderDateTime } from '../../util/time.ts';
import { ContainerWithHeaderAndButtons } from '../ContainerWithHeaderAndButtons';
import { ModalConfirm, ModalSMGroupImmunityEditor } from '../modal';
import { FullTable } from './FullTable';
import { TableCellString } from './TableCellString';

export const SMImmunityTable = ({
    immunities,
    groups,
    isLoading
}: {
    immunities: SMGroupImmunity[];
    groups: SMGroups[];
    isLoading: boolean;
}) => {
    const { sendFlash, sendError } = useUserFlashCtx();
    const queryClient = useQueryClient();
    const [pagination, setPagination] = useState(initPagination(0, RowsPerPage.Ten));

    const onCreateImmunity = async () => {
        try {
            const immunity = await NiceModal.show<SMGroupImmunity>(ModalSMGroupImmunityEditor, { groups });
            queryClient.setQueryData(['serverImmunities'], [...(immunities ?? []), immunity]);
            sendFlash('success', `Group immunity created successfully: ${immunity.group_immunity_id}`);
        } catch (e) {
            logErr(e);
            sendFlash('error', 'Error trying to add group immunity');
        }
    };

    const delImmunityMutation = useMutation({
        mutationKey: ['delGroupImmunity'],
        mutationFn: async ({ immunity }: { immunity: SMGroupImmunity }) => {
            await apiDeleteSMGroupImmunity(immunity.group_immunity_id);
            return immunity;
        },
        onSuccess: (deleted) => {
            queryClient.setQueryData(
                ['serverImmunities'],
                (immunities ?? []).filter((o) => {
                    return o.group_immunity_id != deleted.group_immunity_id;
                })
            );
            sendFlash('success', `Group immunity deleted successfully: ${deleted.group_immunity_id}`);
        },
        onError: sendError
    });

    const immunityColumns = useMemo(() => {
        const onDelete = async (immunity: SMGroupImmunity) => {
            try {
                const confirmed = await NiceModal.show<boolean>(ModalConfirm, {
                    title: 'Delete group immunity?',
                    children: 'This cannot be undone'
                });
                if (!confirmed) {
                    return;
                }
                delImmunityMutation.mutate({ immunity });
            } catch (e) {
                sendFlash('error', `Failed to create confirmation modal: ${e}`);
            }
        };

        return makeGroupImmunityColumns(onDelete);
    }, [delImmunityMutation, sendFlash]);

    return (
        <ContainerWithHeaderAndButtons
            title={'Group Immunities'}
            iconLeft={<FlakyIcon />}
            buttons={[
                <ButtonGroup key={`immunity-header-buttons`} variant={'contained'}>
                    <Button
                        color={'success'}
                        startIcon={<AssuredWorkloadIcon />}
                        onClick={onCreateImmunity}
                        disabled={groups.length < 2}
                    >
                        Add Immunity
                    </Button>
                </ButtonGroup>
            ]}
        >
            <FullTable
                data={immunities ?? []}
                isLoading={isLoading}
                columns={immunityColumns}
                pagination={pagination}
                setPagination={setPagination}
                toOptions={{ from: Route.fullPath }}
            />
        </ContainerWithHeaderAndButtons>
    );
};

const groupImmunityColumnHelper = createColumnHelper<SMGroupImmunity>();

const makeGroupImmunityColumns = (onDelete: (immunity: SMGroupImmunity) => Promise<void>) => [
    groupImmunityColumnHelper.accessor('group.name', {
        header: 'Group',
        size: 500,
        cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    }),
    groupImmunityColumnHelper.accessor('other.name', {
        header: 'Immunity From',
        size: 200,
        cell: (info) => <TableCellString>{info.getValue()}</TableCellString>
    }),
    groupImmunityColumnHelper.accessor('created_on', {
        header: 'Created On',
        size: 140,
        cell: (info) => <TableCellString>{renderDateTime(info.getValue())}</TableCellString>
    }),
    groupImmunityColumnHelper.display({
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
