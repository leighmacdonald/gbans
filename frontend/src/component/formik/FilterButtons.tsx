import React from 'react';
import CheckIcon from '@mui/icons-material/Check';
import Stack from '@mui/material/Stack';
import { ResetButton, SubmitButton } from '../modal/Buttons';

export const FilterButtons = () => {
    return (
        <Stack direction={'row'} spacing={2} flexDirection={'row-reverse'}>
            <SubmitButton label={'Apply'} startIcon={<CheckIcon />} />
            <ResetButton />
        </Stack>
    );
};
