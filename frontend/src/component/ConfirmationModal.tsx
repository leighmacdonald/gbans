import React from 'react';
import ButtonGroup from '@mui/material/ButtonGroup';
import Stack from '@mui/material/Stack';
import Button from '@mui/material/Button';
import CheckIcon from '@mui/icons-material/Check';
import ClearIcon from '@mui/icons-material/Clear';
import { Dialog } from '@mui/material';

export interface ConfirmationModalProps {
    children?: JSX.Element;
    onSuccess?: () => void;
    onCancel?: () => void;
    onOpen?: () => void;
    open: boolean;
    setOpen: (openState: boolean) => void;
}

export const ConfirmationModal = ({
    children,
    open,
    setOpen
}: ConfirmationModalProps) => {
    return (
        <Dialog
            open={open}
            onClose={() => {
                setOpen(false);
            }}
        >
            <Stack>
                {children}
                <ButtonGroup>
                    <Button
                        variant={'contained'}
                        color={'success'}
                        startIcon={<CheckIcon />}
                    >
                        Accept
                    </Button>
                    <Button
                        variant={'contained'}
                        color={'error'}
                        startIcon={<ClearIcon />}
                    >
                        Cancel
                    </Button>
                </ButtonGroup>
            </Stack>
        </Dialog>
    );
};
