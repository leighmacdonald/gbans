import React from 'react';
import ButtonGroup from '@mui/material/ButtonGroup';
import Stack from '@mui/material/Stack';
import Button from '@mui/material/Button';
import CheckIcon from '@mui/icons-material/Check';
import ClearIcon from '@mui/icons-material/Clear';
import { Dialog } from '@mui/material';

export interface ConfirmationModalProps<T> {
    children?: JSX.Element;
    onSuccess?: (resp: T) => void;
    onCancel?: () => void;
    onAccept?: () => void;
    onOpen?: () => void;
    open: boolean;
    setOpen: (openState: boolean) => void;
}

export const ConfirmationModal = ({
    children,
    open,
    setOpen,
    onAccept,
    onCancel
}: ConfirmationModalProps<boolean>) => {
    return (
        <Dialog
            open={open}
            onClose={() => {
                setOpen(false);
            }}
        >
            <Stack padding={2} spacing={2}>
                {children}
                <Stack direction={'row-reverse'}>
                    <ButtonGroup>
                        {onAccept && (
                            <Button
                                variant={'contained'}
                                color={'success'}
                                startIcon={<CheckIcon />}
                                onClick={onAccept}
                            >
                                Accept
                            </Button>
                        )}
                        {onCancel && (
                            <Button
                                variant={'contained'}
                                color={'error'}
                                startIcon={<ClearIcon />}
                                onClick={onCancel}
                            >
                                Cancel
                            </Button>
                        )}
                    </ButtonGroup>
                </Stack>
            </Stack>
        </Dialog>
    );
};
