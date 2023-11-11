import React from 'react';
import CheckIcon from '@mui/icons-material/Check';
import ButtonGroup from '@mui/material/ButtonGroup';
import { ResetButton, SubmitButton } from '../modal/Buttons';

export const FilterButtons = () => {
    return (
        <ButtonGroup>
            <ResetButton />
            <SubmitButton label={'Apply'} startIcon={<CheckIcon />} />
        </ButtonGroup>
    );
};
