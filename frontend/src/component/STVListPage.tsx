import React from 'react';
import VideocamIcon from '@mui/icons-material/Videocam';
import Stack from '@mui/material/Stack';
import { ContainerWithHeader } from './ContainerWithHeader';
import { STVTable } from './STVTable';

export const STVListPage = () => {
    return (
        <Stack spacing={4}>
            <ContainerWithHeader
                title={'SourceTV Recordings'}
                iconLeft={<VideocamIcon />}
            >
                <STVTable />
            </ContainerWithHeader>
        </Stack>
    );
};
