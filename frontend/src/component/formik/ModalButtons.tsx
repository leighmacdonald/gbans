import Button from '@mui/material/Button';
import CheckIcon from '@mui/icons-material/Check';
import ClearIcon from '@mui/icons-material/Clear';
import React from 'react';
import { DialogActions } from '@mui/material';

interface ModalButtonsProps {
    formId: string;
    setOpen: (closed: boolean) => void;
}

export const ModalButtons = ({ formId, setOpen }: ModalButtonsProps) => {
    return (
        <DialogActions>
            <Button
                variant={'contained'}
                color={'success'}
                startIcon={<CheckIcon />}
                type={'submit'}
                form={formId}
            >
                Accept
            </Button>

            <Button
                variant={'contained'}
                color={'warning'}
                startIcon={<ClearIcon />}
                type={'reset'}
                form={formId}
            >
                Reset
            </Button>
            <Button
                variant={'contained'}
                color={'error'}
                startIcon={<ClearIcon />}
                type={'button'}
                form={formId}
                onClick={() => {
                    setOpen(false);
                }}
            >
                Cancel
            </Button>
        </DialogActions>
    );
};
