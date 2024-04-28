import VideocamIcon from '@mui/icons-material/Videocam';
import { createFileRoute } from '@tanstack/react-router';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { STVTable } from '../component/table/STVTable';

export const Route = createFileRoute('/_guest/stv')({
    component: STV
});

function STV() {
    return (
        <ContainerWithHeader title={'SourceTV Recordings'} iconLeft={<VideocamIcon />}>
            <STVTable />
        </ContainerWithHeader>
    );
}
