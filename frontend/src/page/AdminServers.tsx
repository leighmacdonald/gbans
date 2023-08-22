import React, { useCallback, useEffect, useState } from 'react';
import Grid from '@mui/material/Unstable_Grid2';
import Paper from '@mui/material/Paper';
import { DataTable } from '../component/DataTable';
import { apiGetServersAdmin, Server } from '../api';
import { ServerEditorModal } from '../component/ServerEditorModal';
import { Nullable } from '../util/types';
import ButtonGroup from '@mui/material/ButtonGroup';
import IconButton from '@mui/material/IconButton';
import Tooltip from '@mui/material/Tooltip';
import EditIcon from '@mui/icons-material/Edit';
import Button from '@mui/material/Button';
import CreateIcon from '@mui/icons-material/Create';
import Stack from '@mui/material/Stack';
import DeleteIcon from '@mui/icons-material/Delete';
import { DeleteServerModal } from '../component/DeleteServerModal';

export const AdminServers = () => {
    const [open, setOpen] = useState(false);
    const [deleteModalOpen, setDeleteModalOpen] = useState(false);
    const [servers, setServers] = useState<Server[]>([]);
    const [selectedServer, setSelectedServer] = useState<Nullable<Server>>();
    const [isLoading, setIsLoading] = useState(false);

    const reload = useCallback(() => {
        setIsLoading(true);
        apiGetServersAdmin().then((s) => {
            setServers(s.result || []);
            setIsLoading(false);
        });
    }, []);

    useEffect(() => {
        reload();
    }, [reload]);

    return (
        <Grid container spacing={2}>
            <Grid xs={12}>
                <ServerEditorModal
                    setOpen={setOpen}
                    open={open}
                    server={selectedServer}
                    onSuccess={reload}
                />
                {selectedServer && (
                    <DeleteServerModal
                        server={selectedServer}
                        open={deleteModalOpen}
                        setOpen={setDeleteModalOpen}
                        onSuccess={reload}
                    />
                )}
                <Stack spacing={2}>
                    <ButtonGroup>
                        <Button
                            variant={'contained'}
                            color={'secondary'}
                            startIcon={<CreateIcon />}
                            sx={{ marginRight: 2 }}
                            onClick={() => {
                                setSelectedServer(null);
                                setOpen(true);
                            }}
                        >
                            Create Server
                        </Button>
                    </ButtonGroup>
                    <Paper elevation={1}>
                        <DataTable
                            isLoading={isLoading}
                            rowsPerPage={100}
                            defaultSortColumn={'server_name'}
                            rows={servers}
                            columns={[
                                {
                                    tooltip: 'Name',
                                    label: 'Name',
                                    sortKey: 'server_name',
                                    align: 'left',
                                    sortable: true
                                },
                                {
                                    tooltip: 'Name Long',
                                    label: 'Name Long',
                                    sortKey: 'server_name_long',
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
                                // {
                                //     tooltip: 'Password',
                                //     label: 'Password',
                                //     sortKey: 'password',
                                //     align: 'left'
                                // },
                                {
                                    tooltip: 'Region',
                                    label: 'Region',
                                    sortKey: 'region',
                                    align: 'left'
                                },
                                {
                                    tooltip: 'CC',
                                    label: 'CC',
                                    sortKey: 'cc',
                                    align: 'left'
                                },
                                // {
                                //     tooltip: 'Location',
                                //     label: 'Location',
                                //     virtual: true,
                                //     virtualKey: 'location',
                                //     align: 'left'
                                // },
                                {
                                    tooltip: 'Enabled',
                                    label: 'En.',
                                    sortKey: 'is_enabled',
                                    sortable: true,
                                    align: 'left',
                                    renderer: (row) =>
                                        row.is_enabled ? 'On' : 'Off'
                                },
                                {
                                    label: 'Act.',
                                    tooltip: 'Actions',
                                    sortable: false,
                                    align: 'right',
                                    renderer: (row) => (
                                        <ButtonGroup fullWidth>
                                            <IconButton
                                                color={'warning'}
                                                onClick={() => {
                                                    setSelectedServer(row);
                                                    setOpen(true);
                                                }}
                                            >
                                                <Tooltip title={'Edit Server'}>
                                                    <EditIcon />
                                                </Tooltip>
                                            </IconButton>
                                            <IconButton
                                                color={'warning'}
                                                onClick={() => {
                                                    setSelectedServer(row);
                                                    setDeleteModalOpen(true);
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
                    </Paper>
                </Stack>
            </Grid>
        </Grid>
    );
};
