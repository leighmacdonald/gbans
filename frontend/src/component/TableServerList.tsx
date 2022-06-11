import { apiGetServers, ServerState } from '../api';
import { CreateDataTable } from './DataTable';

export const TableServerList = (): JSX.Element =>
    CreateDataTable<ServerState>()({
        connector: async () => {
            return (await apiGetServers()) || [];
        },
        id_field: 'server_id',
        heading: 'Servers',
        headers: [
            // {id: "server_id",disablePadding: true, label: "ID", numeric: true},
            {
                id: 'name_short',
                disablePadding: false,
                label: 'Name',
                cell_type: 'string'
            },
            {
                id: 'name',
                disablePadding: false,
                label: 'Name Long',
                cell_type: 'string'
            },
            {
                id: 'host',
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
                id: 'password',
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
                id: 'location',
                disablePadding: false,
                label: 'Lat',
                cell_type: 'number'
            }
        ],
        showToolbar: true
    });
