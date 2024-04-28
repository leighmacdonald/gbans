import CellTowerIcon from '@mui/icons-material/CellTower';
import { createFileRoute } from '@tanstack/react-router';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { NetworkInfo } from '../component/NetworkInfo.tsx';

export const Route = createFileRoute('/_mod/admin/network/ip_Info')({
    component: AdminNetworkInfo
});

function AdminNetworkInfo() {
    return (
        <ContainerWithHeader title="Network Info" iconLeft={<CellTowerIcon />}>
            <NetworkInfo />
        </ContainerWithHeader>
    );
}
