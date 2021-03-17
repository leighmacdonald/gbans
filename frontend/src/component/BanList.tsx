import { apiGetBans, IAPIBanRecord } from '../util/api';

import { CreateDataTable } from './DataTable';

export const BanList = (): JSX.Element =>
    CreateDataTable<IAPIBanRecord>()({
        connector: async () => {
            return (await apiGetBans()) as Promise<IAPIBanRecord[]>;
        },
        id_field: 'ban_id',
        heading: 'Bans',
        headers: [
            {
                id: 'personaname',
                disablePadding: false,
                label: 'Name',
                numeric: false
            },
            {
                id: 'reason_text',
                disablePadding: false,
                label: 'Reason',
                numeric: false
            },
            {
                id: 'ip_addr',
                disablePadding: false,
                label: 'IP/Net',
                numeric: false
            },
            {
                id: 'valid_until',
                disablePadding: false,
                label: 'Valid Until',
                numeric: false
            },
            {
                id: 'realname',
                disablePadding: false,
                label: 'Real Name',
                numeric: false
            },
            {
                id: 'source',
                disablePadding: false,
                label: 'Source',
                numeric: false
            }
        ],
        showToolbar: false
    });
