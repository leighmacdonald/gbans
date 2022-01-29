import { apiGetServers, Server } from '../api';
import { CreateDataTable } from './DataTable';

export const ServerList = (): JSX.Element =>
    CreateDataTable<Server>()({
        connector: async () => {
            return (await apiGetServers()) || [];
        },
        id_field: 'server_id',
        heading: 'Servers',
        headers: [
            // {id: "server_id",disablePadding: true, label: "ID", numeric: true},
            {
                id: 'server_name',
                disablePadding: false,
                label: 'Name',
                cell_type: 'string'
            },
            {
                id: 'server_name_long',
                disablePadding: false,
                label: 'Name Long',
                cell_type: 'string'
            },
            {
                id: 'address',
                disablePadding: false,
                label: 'Host',
                cell_type: 'string'
            },
            {
                id: 'port',
                disablePadding: false,
                label: 'Port',
                cell_type: 'number'
            },
            {
                id: 'password_protected',
                disablePadding: false,
                label: 'Private',
                cell_type: 'bool'
            },
            {
                id: 'region',
                disablePadding: false,
                label: 'Region',
                cell_type: 'string'
            },
            {
                id: 'cc',
                disablePadding: false,
                label: 'Country',
                cell_type: 'flag'
            },
            {
                id: 'latitude',
                disablePadding: false,
                label: 'Lat',
                cell_type: 'number'
            },
            {
                id: 'longitude',
                disablePadding: false,
                label: 'Lon',
                cell_type: 'number'
            }
        ],
        showToolbar: true
    });
