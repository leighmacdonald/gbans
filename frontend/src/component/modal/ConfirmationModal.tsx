import React, { JSX } from 'react';
import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import {
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle
} from '@mui/material';
import { Breakpoint } from '@mui/material';
import { CancelButton, ConfirmButton } from './Buttons';

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
    ({
        children,
        onAccept,
        onCancel,
        title,
        size,
        fullWidth
    }: ConfirmationModalProps<boolean>) => {
        const modal = useModal();
        return (
            <Dialog
                fullWidth={fullWidth}
                maxWidth={size ?? 'xl'}
                {...muiDialogV5(modal)}
            >
                {title && <DialogTitle>{title}</DialogTitle>}

                <DialogContent>{children}</DialogContent>
                <DialogActions>
                    <CancelButton
                        onClick={async () => {
                            if (onCancel != undefined) {
                                onCancel();
                            }
                            modal.resolve(false);
                            await modal.hide();
                        }}
                    />

                    <ConfirmButton
                        onClick={async () => {
                            if (onAccept != undefined) {
                                onAccept();
                            }
                            modal.resolve(true);
                            await modal.hide();
                        }}
                    />
                </DialogActions>
            </Dialog>
        );
    }
);

export default ConfirmationModal;
