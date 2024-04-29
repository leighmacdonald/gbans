import WifiOffIcon from '@mui/icons-material/WifiOff';
import { createFileRoute } from '@tanstack/react-router';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { NetworkBlockSources } from '../component/NetworkBlockSources.tsx';

export const Route = createFileRoute('/_mod/admin/network/cidrblocks')({
    component: AdminNetworkCIDRBlocks
});

function AdminNetworkCIDRBlocks() {
    return (
        <ContainerWithHeader title="Admin Network CIDR" iconLeft={<WifiOffIcon />}>
            <NetworkBlockSources />
        </ContainerWithHeader>
    );
}
