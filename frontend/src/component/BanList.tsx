import { apiGetBans, IAPIBanRecord } from '../api';

import { CreateDataTable } from './DataTable';

export const BanList = (): JSX.Element =>
    CreateDataTable<IAPIBanRecord>()({
        connector: async () => {
            return await apiGetBans({
                order_by: 'ban_id',
                desc: true
            });
        },
        id_field: 'ban_id',
        heading: 'Bans',
        headers: [
            {
                id: 'personaname',
                disablePadding: false,
                label: 'Name',
                cell_type: 'string'
            },
            {
                id: 'reason_text',
                disablePadding: false,
                label: 'Reason',
                cell_type: 'string'
            },
            {
                id: 'ip_addr',
                disablePadding: false,
                label: 'IP/Net',
                cell_type: 'string'
            },
            {
                id: 'valid_until',
                disablePadding: false,
                label: 'Valid Until',
                cell_type: 'date'
            },
            {
                id: 'realname',
                disablePadding: false,
                label: 'Real Name',
                cell_type: 'string'
            },
            {
                id: 'source',
                disablePadding: false,
                label: 'Source',
                cell_type: 'string'
            }
        ],
        showToolbar: true
    });
