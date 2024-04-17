import WifiFindIcon from '@mui/icons-material/WifiFind';
import { ContainerWithHeader } from '../component/ContainerWithHeader.tsx';
import { FindPlayesrByCIDR } from '../component/FindPlayesrByCIDR.tsx';

export const AdminNetworkPlayersByCIDRPage = () => {
    return (
        <ContainerWithHeader
            title={'Find Players By IP/CIDR'}
            iconLeft={<WifiFindIcon />}
        >
            <FindPlayesrByCIDR />
        </ContainerWithHeader>
    );
};

export default AdminNetworkPlayersByCIDRPage;
