import SensorOccupiedIcon from '@mui/icons-material/SensorOccupied';
import { createLazyFileRoute } from '@tanstack/react-router';
import { ContainerWithHeader } from '../../component/ContainerWithHeader';
import { FindPlayerIPs } from '../../component/FindPlayerIPs';

export const Route = createLazyFileRoute('/_auth/admin/network/ip_hist')({
    component: AdminNetworkPlayerIPHistory
});

function AdminNetworkPlayerIPHistory() {
    return (
        <ContainerWithHeader title="Player IP History" iconLeft={<SensorOccupiedIcon />}>
            <FindPlayerIPs />
        </ContainerWithHeader>
    );
}
