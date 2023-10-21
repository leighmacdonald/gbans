import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import CheckIcon from '@mui/icons-material/Check';
import {
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle
} from '@mui/material';
import Stack from '@mui/material/Stack';
import Typography from '@mui/material/Typography';
import { useTheme } from '@mui/material/styles';
import React, { useCallback, useState } from 'react';
import { Heading } from '../Heading';
import { LoadingSpinner } from '../LoadingSpinner';
import { CancelButton, ConfirmButton } from './Buttons';

export interface ConfirmProps {
    title?: string;
    description?: string;
    onError?: (error: string) => void;
    onConfirm: () => Promise<void>;
}

export const Confirm = NiceModal.create(
    ({ title, description, onConfirm }: ConfirmProps) => {
        const [inProgress, setInProgres] = useState(false);
        const [error, setError] = useState('');
        const modal = useModal();
        const theme = useTheme();

        const defaultStartDate = new Date();
        const defaultEndDate = new Date();
        defaultEndDate.setDate(defaultStartDate.getDate() + 1);

        const onSubmit = useCallback(async () => {
            setInProgres(true);
            try {
                await onConfirm();
            } catch (e) {
                setError(`${e}`);
            } finally {
                setInProgres(false);
            }
        }, [onConfirm]);

        const formId = 'confirmForm';

        return (
            <form id={formId}>
                <Dialog fullWidth {...muiDialogV5(modal)}>
                    <DialogTitle
                        component={Heading}
                        iconLeft={
                            inProgress ? <LoadingSpinner /> : <CheckIcon />
                        }
                    >
                        {title ?? 'Confirmation'}
                    </DialogTitle>

                    <DialogContent>
                        <Stack spacing={2}>
                            <Typography variant={'body1'}>
                                {description ?? 'Are you sure?'}
                            </Typography>
                            {error != '' && (
                                <Typography
                                    variant={'body1'}
                                    sx={{ color: theme.palette.error.main }}
                                >
                                    {error}
                                </Typography>
                            )}
                        </Stack>
                    </DialogContent>
                    <DialogActions>
                        <CancelButton onClick={modal.hide} />
                        <ConfirmButton
                            onClick={onSubmit}
                            disabled={error != ''}
                        />
                    </DialogActions>
                </Dialog>
            </form>
        );
    }
);
