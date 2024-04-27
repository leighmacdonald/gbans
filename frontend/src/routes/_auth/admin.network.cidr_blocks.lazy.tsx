import WifiOffIcon from '@mui/icons-material/WifiOff';
import { createLazyFileRoute } from '@tanstack/react-router';
import { ContainerWithHeader } from '../../component/ContainerWithHeader.tsx';
import { NetworkBlockSources } from '../../component/NetworkBlockSources.tsx';

export const Route = createLazyFileRoute('/_auth/admin/network/cidr_blocks')({
    component: AdminNetworkCIDRBlocks
});

function AdminNetworkCIDRBlocks() {
    return (
        <ContainerWithHeader title="Admin Network CIDR" iconLeft={<WifiOffIcon />}>
            <NetworkBlockSources />
        </ContainerWithHeader>
    );
}
