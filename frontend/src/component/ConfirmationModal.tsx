import React, { JSX } from 'react';
import CheckIcon from '@mui/icons-material/Check';
import ClearIcon from '@mui/icons-material/Clear';
import {
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle
} from '@mui/material';
import Button from '@mui/material/Button';
import { Breakpoint } from '@mui/system';

export interface ConfirmationModalProps<T> {
    initialValue?: T;
    children?: JSX.Element;
    onSuccess?: (resp: T) => void;
    onCancel?: () => void;
    onAccept?: () => void;
    onOpen?: () => void;
    open: boolean;
    setOpen: (openState: boolean) => void;
    title?: string;
    size?: Breakpoint;
    fullWidth?: boolean;
}

export const ConfirmationModal = ({
    children,
    open,
    setOpen,
    onAccept,
    onCancel,
    title,
    size,
    fullWidth
}: ConfirmationModalProps<boolean>) => {
    return (
        <Dialog
            fullWidth={fullWidth}
            maxWidth={size ?? 'xl'}
            open={open}
            onClose={() => {
                setOpen(false);
            }}
        >
            {title && <DialogTitle>{title}</DialogTitle>}

            <DialogContent>{children}</DialogContent>
            <DialogActions>
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
            </DialogActions>
        </Dialog>
    );
};
