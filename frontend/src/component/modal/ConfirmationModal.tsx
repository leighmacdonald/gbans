import { JSX } from 'react';
import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import CheckIcon from '@mui/icons-material/Check';
import CloseIcon from '@mui/icons-material/Close';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import { Breakpoint } from '@mui/material';
import Button from '@mui/material/Button';
import { ModalConfirm } from './index.ts';

export interface ConfirmationModalProps<T> {
    initialValue?: T;
    children?: JSX.Element;
    onSuccess?: (resp: T) => void;
    onCancel?: () => void;
    onAccept?: () => void;
    title?: string;
    size?: Breakpoint;
    fullWidth?: boolean;
}

export const ConfirmationModal = NiceModal.create(
    ({ children, title, size, fullWidth }: ConfirmationModalProps<boolean>) => {
        const modal = useModal(ModalConfirm);

        const accept = async () => {
            modal.resolve(true);
            await modal.hide();
        };

        const cancel = async () => {
            modal.resolve(false);
            await modal.hide();
        };

        return (
            <Dialog fullWidth={fullWidth} maxWidth={size ?? 'xl'} {...muiDialogV5(modal)}>
                {title && <DialogTitle>{title}</DialogTitle>}

                <DialogContent>{children}</DialogContent>
                <DialogActions>
                    <Button variant={'contained'} color={'success'} startIcon={<CheckIcon />} onClick={accept}>
                        Accept
                    </Button>
                    <Button variant={'contained'} color={'error'} startIcon={<CloseIcon />} onClick={cancel}>
                        Cancel
                    </Button>
                </DialogActions>
            </Dialog>
        );
    }
);
