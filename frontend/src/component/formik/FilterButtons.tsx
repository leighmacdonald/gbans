import React from 'react';
import CheckIcon from '@mui/icons-material/Check';
import ButtonGroup from '@mui/material/ButtonGroup';
import { VCenterBox } from '../VCenterBox';
import { ResetButton, SubmitButton } from '../modal/Buttons';

export const FilterButtons = () => {
    return (
        <VCenterBox>
            <ButtonGroup fullWidth>
                <ResetButton />
                <SubmitButton label={'Apply'} startIcon={<CheckIcon />} />
            </ButtonGroup>
        </VCenterBox>
    );
};
