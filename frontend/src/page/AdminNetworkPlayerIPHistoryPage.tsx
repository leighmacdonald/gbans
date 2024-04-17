import SensorOccupiedIcon from '@mui/icons-material/SensorOccupied';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { FindPlayerIPs } from '../component/FindPlayerIPs';

export const AdminNetworkPlayerIPHistoryPage = () => {
    return (
        <ContainerWithHeader
            title="Player IP History"
            iconLeft={<SensorOccupiedIcon />}
        >
            <FindPlayerIPs />
        </ContainerWithHeader>
    );
};

export default AdminNetworkPlayerIPHistoryPage;
