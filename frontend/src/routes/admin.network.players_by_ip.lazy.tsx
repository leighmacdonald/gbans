import WifiFindIcon from '@mui/icons-material/WifiFind';
import { createLazyFileRoute } from '@tanstack/react-router';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { FindPlayesrByCIDR } from '../component/FindPlayesrByCIDR.tsx';

export const Route = createLazyFileRoute('/admin/network/players_by_ip')({
    component: AdminNetworkPlayersByCIDR
});

function AdminNetworkPlayersByCIDR() {
    return (
        <ContainerWithHeader
            title={'Find Players By IP/CIDR'}
            iconLeft={<WifiFindIcon />}
        >
            <FindPlayesrByCIDR />
        </ContainerWithHeader>
    );
}
