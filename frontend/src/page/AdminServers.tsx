import React, { useCallback, useEffect, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import CreateIcon from '@mui/icons-material/Create';
import DeleteIcon from '@mui/icons-material/Delete';
import EditIcon from '@mui/icons-material/Edit';
import Button from '@mui/material/Button';
import ButtonGroup from '@mui/material/ButtonGroup';
import IconButton from '@mui/material/IconButton';
import Paper from '@mui/material/Paper';
import Stack from '@mui/material/Stack';
import Tooltip from '@mui/material/Tooltip';
import Grid from '@mui/material/Unstable_Grid2';
import { noop } from 'lodash-es';
import { apiGetServersAdmin, Server } from '../api';
import { DataTable } from '../component/DataTable';
import { ModalServerDelete, ModalServerEditor } from '../component/modal';
import { ServerEditorModal } from '../component/modal/ServerEditorModal';
import { logErr } from '../util/errors';

export const AdminServers = () => {
    const [servers, setServers] = useState<Server[]>([]);
    const [isLoading, setIsLoading] = useState(false);

    const reload = useCallback(() => {
        const abortController = new AbortController();

        const fetchServers = async () => {
            try {
                setIsLoading(true);
                setServers((await apiGetServersAdmin(abortController)) || []);
            } catch (e) {
                logErr(e);
            } finally {
                setIsLoading(false);
            }
        };

        fetchServers().then(noop);

        return () => abortController.abort();
    }, []);

    useEffect(() => {
        reload();
    }, [reload]);

    return (
        <Grid container spacing={2}>
            <Grid xs={12}>
                <Stack spacing={2}>
                    <ButtonGroup>
                        <Button
                            variant={'contained'}
                            color={'secondary'}
                            startIcon={<CreateIcon />}
                            sx={{ marginRight: 2 }}
                            onClick={async () => {
                                await NiceModal.show(ServerEditorModal, {});
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
                                                    await NiceModal.show(
                                                        ModalServerDelete,
                                                        { server: row }
                                                    );
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
