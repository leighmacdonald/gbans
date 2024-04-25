import CellTowerIcon from '@mui/icons-material/CellTower';
import { createLazyFileRoute } from '@tanstack/react-router';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { NetworkInfo } from '../component/NetworkInfo.tsx';

export const Route = createLazyFileRoute('/admin/network/ip_Info')({
    component: AdminNetworkInfo
});

export const AdminNetworkInfo = () => {
    return (
        <ContainerWithHeader title="Network Info" iconLeft={<CellTowerIcon />}>
            <NetworkInfo />
        </ContainerWithHeader>
    );
};
