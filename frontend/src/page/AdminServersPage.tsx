import React, { useCallback, useMemo, useState } from 'react';
import useUrlState from '@ahooksjs/use-url-state';
import NiceModal from '@ebay/nice-modal-react';
import AddIcon from '@mui/icons-material/Add';
import DeleteIcon from '@mui/icons-material/Delete';
import EditIcon from '@mui/icons-material/Edit';
import StorageIcon from '@mui/icons-material/Storage';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import IconButton from '@mui/material/IconButton';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Grid from '@mui/material/Unstable_Grid2';
import { Server } from '../api';
import { ContainerWithHeaderAndButtons } from '../component/ContainerWithHeaderAndButtons';
import { ModalServerEditor } from '../component/modal';
import { LazyTable, RowsPerPage } from '../component/table/LazyTable';
import { TableCellBool } from '../component/table/TableCellBool';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { useServersAdmin } from '../hooks/useServersAdmin';

export const AdminServersPage = () => {
    const [newServers, setNewServers] = useState<Server[]>([]);
    const [state, setState] = useUrlState({
        page: undefined,
        deleted: undefined,
        rows: undefined,
        sortOrder: undefined,
        sortColumn: undefined
    });
    const { sendFlash } = useUserFlashCtx();
    const { data, count } = useServersAdmin({
        limit: Number(state.rows ?? RowsPerPage.TwentyFive),
        offset: Number(
            (state.page ?? 0) * (state.rows ?? RowsPerPage.TwentyFive)
        ),
        order_by: state.sortColumn ?? 'short_name',
        desc: state.sortOrder == 'desc',
        deleted: false,
        include_disabled: true
    });

    const servers = useMemo(() => {
        return [...newServers, ...data];
    }, [data, newServers]);

    const onCreate = useCallback(async () => {
        try {
            const newServer = await NiceModal.show<Server>(
                ModalServerEditor,
                {}
            );
            setNewServers((prevState) => {
                return [newServer, ...prevState];
            });

            sendFlash('success', 'Server created successfully');
        } catch (e) {
            sendFlash('error', `Failed to create new server: ${e}`);
        }
    }, [sendFlash]);

    const onEdit = useCallback(
        async (server: Server) => {
            try {
                const newServer = await NiceModal.show<Server>(
                    ModalServerEditor,
                    { server }
                );
                setNewServers((prevState) => {
                    return [newServer, ...prevState];
                });

                sendFlash('success', 'Server edited successfully');
            } catch (e) {
                sendFlash('error', `Failed to edit server: ${e}`);
            }
        },
        [sendFlash]
    );

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
                        <LazyTable<Server>
                            showPager={true}
                            count={count}
                            rows={servers}
                            page={Number(state.page ?? 0)}
                            rowsPerPage={Number(
                                state.rows ?? RowsPerPage.TwentyFive
                            )}
                            sortOrder={state.sortOrder}
                            sortColumn={state.sortColumn}
                            onSortColumnChanged={async (column) => {
                                setState({ sortColumn: column });
                            }}
                            onSortOrderChanged={async (direction) => {
                                setState({ sortOrder: direction });
                            }}
                            onPageChange={(_, newPage: number) => {
                                setState({ page: newPage });
                            }}
                            onRowsPerPageChange={(
                                event: React.ChangeEvent<
                                    HTMLInputElement | HTMLTextAreaElement
                                >
                            ) => {
                                setState({
                                    rows: Number(event.target.value),
                                    page: 0
                                });
                            }}
                            columns={[
                                {
                                    tooltip: 'Name',
                                    label: 'Name',
                                    sortKey: 'short_name',
                                    align: 'left',
                                    sortable: true
                                },
                                {
                                    tooltip: 'Name Long',
                                    label: 'Name Long',
                                    sortKey: 'name',
                                    align: 'left',
                                    sortable: true
                                },
                                {
                                    tooltip: 'Address',
                                    label: 'Address',
                                    sortKey: 'address',
                                    align: 'left',
                                    sortable: true
                                },
                                {
                                    tooltip: 'Port',
                                    label: 'Port',
                                    sortKey: 'port',
                                    align: 'left',
                                    sortable: true
                                },
                                {
                                    tooltip: 'RCON Password',
                                    label: 'rcon',
                                    sortKey: 'rcon',
                                    align: 'left'
                                },
                                {
                                    tooltip: 'Region',
                                    label: 'Region',
                                    sortKey: 'region',
                                    align: 'left',
                                    sortable: true
                                },
                                {
                                    tooltip: 'CC',
                                    label: 'CC',
                                    sortKey: 'cc',
                                    align: 'left',
                                    sortable: true
                                },
                                {
                                    tooltip: 'Stats Recording Enabled',
                                    label: 'Stats',
                                    sortKey: 'enable_stats',
                                    align: 'left',
                                    sortable: true,
                                    renderer: (row) => {
                                        return (
                                            <TableCellBool
                                                enabled={row.enable_stats}
                                            />
                                        );
                                    }
                                },
                                {
                                    tooltip: 'Enabled',
                                    label: 'En.',
                                    sortKey: 'is_enabled',
                                    sortable: true,
                                    align: 'center',
                                    renderer: (row) => (
                                        <TableCellBool
                                            enabled={row.is_enabled}
                                        />
                                    )
                                },
                                {
                                    label: 'Act.',
                                    tooltip: 'Actions',
                                    sortable: false,
                                    align: 'center',
                                    renderer: (row) => (
                                        <ButtonGroup fullWidth>
                                            <IconButton
                                                color={'warning'}
                                                onClick={async () => {
                                                    await NiceModal.show(
                                                        ModalServerEditor,
                                                        {
                                                            server: row
                                                        }
                                                    );
                                                }}
                                            >
                                                <Tooltip title={'Edit Server'}>
                                                    <EditIcon />
                                                </Tooltip>
                                            </IconButton>
                                            <IconButton
                                                color={'warning'}
                                                onClick={async () => {
                                                    await onEdit(row);
                                                }}
                                            >
                                                <Tooltip
                                                    title={'Delete Server'}
                                                >
                                                    <DeleteIcon
                                                        color={'error'}
                                                    />
                                                </Tooltip>
                                            </IconButton>
                                        </ButtonGroup>
                                    )
                                }
                            ]}
                        />
                    </ContainerWithHeaderAndButtons>
                </Stack>
            </Grid>
        </Grid>
    );
};
