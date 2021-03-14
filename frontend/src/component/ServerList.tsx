import {apiGetServers, Server} from '../util/api';
import {createDataTable} from './DataTable';

export const ServerList = () =>
    createDataTable<Server>()({
        connector: async () => {
            return (await apiGetServers()) as Promise<Server[]>;
        },
        id_field: 'server_id',
        heading: 'Servers',
        headers: [
            // {id: "server_id",disablePadding: true, label: "ID", numeric: true},
            {
                id: 'server_name',
                disablePadding: false,
                label: 'Name',
                numeric: false
            },
            {id: 'address', disablePadding: false, label: 'Host', numeric: false},
            {id: 'port', disablePadding: false, label: 'Port', numeric: true},
            {id: 'rcon', disablePadding: false, label: 'RCON', numeric: false},
            {
                id: 'token_created_on',
                disablePadding: false,
                label: 'Token Last Updated',
                numeric: false
            }
        ]
    });
