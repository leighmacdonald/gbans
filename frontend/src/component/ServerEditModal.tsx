import { ServerAddForm } from './ServerAddForm';
import { Heading } from './Heading';
import Stack from '@mui/material/Stack';
import React, { useCallback } from 'react';
import { Server } from '../api';
import { ConfirmationModal, ConfirmationModalProps } from './ConfirmationModal';

export const ServerEditModal = ({
    open,
    setOpen
}: ConfirmationModalProps<Server>) => {
    const handleSubmit = useCallback(() => {}, []);
    return (
        <ConfirmationModal
            open={open}
            setOpen={setOpen}
            onSuccess={() => {
                handleSubmit();
            }}
            aria-labelledby="modal-modal-title"
            aria-describedby="modal-modal-description"
        >
            <Stack spacing={2}>
                <Heading>Ban Player</Heading>
                <ServerAddForm />
            </Stack>
        </ConfirmationModal>
    );
};
