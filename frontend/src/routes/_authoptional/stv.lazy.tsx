import VideocamIcon from '@mui/icons-material/Videocam';
import { createLazyFileRoute } from '@tanstack/react-router';
import { ContainerWithHeader } from '../../component/ContainerWithHeader';
import { STVTable } from '../../component/table/STVTable';

export const Route = createLazyFileRoute('/_authoptional/stv')({
    component: STV
});

function STV() {
    return (
        <ContainerWithHeader title={'SourceTV Recordings'} iconLeft={<VideocamIcon />}>
            <STVTable />
        </ContainerWithHeader>
    );
}
