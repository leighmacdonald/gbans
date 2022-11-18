import Button from '@mui/material/Button';
import CheckIcon from '@mui/icons-material/Check';
import ClearIcon from '@mui/icons-material/Clear';
import React from 'react';
import { DialogActions } from '@mui/material';
import { LoadingButton } from '@mui/lab';

interface ModalButtonsProps {
    formId: string;
    setOpen: (closed: boolean) => void;

    inProgress: boolean;
}

export const ModalButtons = ({
    formId,
    setOpen,
    inProgress
}: ModalButtonsProps) => {
    return (
        <DialogActions>
            <LoadingButton
                loading={inProgress}
                variant={'contained'}
                color={'success'}
                startIcon={<CheckIcon />}
                type={'submit'}
                form={formId}
            >
                Accept
            </LoadingButton>

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
