import WifiOffIcon from '@mui/icons-material/WifiOff';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { NetworkBlockSources } from '../component/NetworkBlockSources.tsx';

export const AdminNetworkCIDRBlocksPage = () => {
    return (
        <ContainerWithHeader
            title="Admin Network CIDR"
            iconLeft={<WifiOffIcon />}
        >
            <NetworkBlockSources />
        </ContainerWithHeader>
    );
};

export default AdminNetworkCIDRBlocksPage;
