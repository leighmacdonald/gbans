import Tooltip from '@mui/material/Tooltip';
import Typography from '@mui/material/Typography';
import { PersonConnection } from '../../api';
import { HeadingCell } from '../../util/table.ts';
import { renderDateTime } from '../../util/text.tsx';

export const connectionColumns: HeadingCell<PersonConnection>[] = [
    {
        label: 'Created',
        tooltip: 'Created On',
        sortKey: 'created_on',
        sortType: 'date',
        align: 'left',
        width: '150px',
        sortable: true,
        renderer: (obj: PersonConnection) => (
            <Typography variant={'body1'}>
                {renderDateTime(obj.created_on)}
            </Typography>
        )
    },
    {
        label: 'Name',
        tooltip: 'Name',
        sortKey: 'persona_name',
        sortType: 'string',
        align: 'left',
        width: '150px',
        sortable: true
    },
    {
        label: 'IP Address',
        tooltip: 'IP Address',
        sortKey: 'ip_addr',
        sortType: 'string',
        align: 'left',
        sortable: true
    },
    {
        label: 'Server',
        tooltip: 'IP Address',
        sortKey: 'ip_addr',
        sortType: 'string',
        align: 'left',
        sortable: true,
        renderer: (obj: PersonConnection) => {
            return (
                <Tooltip title={obj.server_name ?? 'Unknown'}>
                    <Typography variant={'body1'}>
                        {obj.server_name_short ?? 'Unknown'}
                    </Typography>
                </Tooltip>
            );
        }
    }
];
