import { apiGetPeople, Person } from '../api';
import { CreateDataTable } from './DataTable';

export const PlayerList = (): JSX.Element =>
    CreateDataTable<Person>()({
        connector: async () => {
            return await apiGetPeople();
        },
        id_field: 'steam_id',
        heading: 'Players',
        headers: [
            // {id: "server_id",disablePadding: true, label: "ID", numeric: true},
            {
                id: 'steam_id',
                disablePadding: false,
                label: 'Steam ID',
                cell_type: 'number'
            },
            {
                id: 'personaname',
                disablePadding: false,
                label: 'Name',
                cell_type: 'string'
            },
            {
                id: 'loccountrycode',
                disablePadding: false,
                label: 'Country',
                cell_type: 'flag'
            },
            {
                id: 'created_on',
                disablePadding: false,
                label: 'Created',
                cell_type: 'date'
            },
            {
                id: 'updated_on',
                disablePadding: false,
                label: 'Updated',
                cell_type: 'date'
            }
        ],
        showToolbar: false
    });
