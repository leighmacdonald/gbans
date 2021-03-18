import { apiGetPeople, Person } from '../util/api';
import { CreateDataTable } from './DataTable';

export const PlayerList = (): JSX.Element =>
    CreateDataTable<Person>()({
        connector: async () => {
            return (await apiGetPeople()) as Promise<Person[]>;
        },
        id_field: 'steam_id',
        heading: 'Players',
        headers: [
            // {id: "server_id",disablePadding: true, label: "ID", numeric: true},
            {
                id: 'steam_id',
                disablePadding: false,
                label: 'Steam ID',
                numeric: true
            },
            {
                id: 'personaname',
                disablePadding: false,
                label: 'Name',
                numeric: false
            },
            {
                id: 'loccountrycode',
                disablePadding: false,
                label: 'Country',
                numeric: false
            },
            {
                id: 'created_on',
                disablePadding: false,
                label: 'Create On',
                numeric: false
            },
            {
                id: 'updated_on',
                disablePadding: false,
                label: 'Last Updated',
                numeric: false
            }
        ],
        showToolbar: false
    });
