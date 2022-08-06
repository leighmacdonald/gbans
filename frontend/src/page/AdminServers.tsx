import React, { useEffect, useState } from 'react';
import Grid from '@mui/material/Grid';
import { ServerAddForm } from '../component/ServerAddForm';
import Paper from '@mui/material/Paper';
import { DataTable } from '../component/DataTable';
import { apiGetServerStates, ServerState } from '../api';

export const AdminServers = (): JSX.Element => {
    const [servers, setServers] = useState<ServerState[]>([]);
    const [isLoading, setIsLoading] = useState(false);
    useEffect(() => {
        setIsLoading(true);
        apiGetServerStates().then((s) => {
            setServers(s);
            setIsLoading(false);
        });
    }, []);

    return (
        <Grid container spacing={3} paddingTop={3}>
            <Grid item xs={12}>
                <Paper elevation={1}>
                    <ServerAddForm />
                </Paper>
            </Grid>
            <Grid item xs={12}>
                <Paper elevation={1}>
                    <DataTable
                        isLoading={isLoading}
                        rowsPerPage={100}
                        defaultSortColumn={'name_short'}
                        rows={servers}
                        columns={[
                            {
                                tooltip: 'Name',
                                label: 'Name',
                                sortKey: 'name_short',
                                align: 'left'
                            },
                            {
                                tooltip: 'Name Long',
                                label: 'Name Long',
                                sortKey: 'name',
                                align: 'left'
                            },
                            {
                                tooltip: 'Address',
                                label: 'Address',
                                sortKey: 'host',
                                align: 'left'
                            },
                            {
                                tooltip: 'Port',
                                label: 'Port',
                                sortKey: 'port',
                                align: 'left'
                            },
                            {
                                tooltip: 'Password',
                                label: 'Password',
                                sortKey: 'password',
                                align: 'left'
                            },
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
                            {
                                tooltip: 'Location',
                                label: 'Location',
                                virtual: true,
                                virtualKey: 'location',
                                align: 'left'
                            }
                        ]}
                    />
                </Paper>
            </Grid>
        </Grid>
    );
};
