import React, { JSX } from 'react';
import VideocamIcon from '@mui/icons-material/Videocam';
import { ContainerWithHeader } from '../component/ContainerWithHeader';
import { STVTable } from '../component/table/STVTable';

export const STVPage = (): JSX.Element => {
    return (
        <ContainerWithHeader
            title={'SourceTV Recordings'}
            iconLeft={<VideocamIcon />}
        >
            <STVTable />
        </ContainerWithHeader>
    );
};

export default STVPage;
