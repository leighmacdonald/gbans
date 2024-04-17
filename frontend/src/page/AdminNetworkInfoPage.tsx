import CellTowerIcon from '@mui/icons-material/CellTower';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { NetworkInfo } from '../component/NetworkInfo.tsx';

export const AdminNetworkInfoPage = () => {
    return (
        <ContainerWithHeader title="Network Info" iconLeft={<CellTowerIcon />}>
            <NetworkInfo />
        </ContainerWithHeader>
    );
};

export default AdminNetworkInfoPage;
