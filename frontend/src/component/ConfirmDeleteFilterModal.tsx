import React, { useCallback } from 'react';
import Stack from '@mui/material/Stack';
import { ConfirmationModal, ConfirmationModalProps } from './ConfirmationModal';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { Heading } from './Heading';
import { apiDeleteFilter, Filter } from '../api/filters';

export interface ConfirmDeleteFilterModalProps
    extends ConfirmationModalProps<Filter> {
    record: Filter;
}

export const ConfirmDeleteFilterModal = ({
    open,
    setOpen,
    onSuccess,
    record
}: ConfirmDeleteFilterModalProps) => {
    const { sendFlash } = useUserFlashCtx();

    const handleSubmit = useCallback(() => {
        apiDeleteFilter(record.word_id)
            .then(() => {
                sendFlash('success', `Deleted filter successfully`);
                onSuccess && onSuccess(record);
            })
            .catch((err) => {
                sendFlash('error', `Failed to delete filter: ${err}`);
            });
    }, [record, sendFlash, onSuccess]);

    return (
        <ConfirmationModal
            open={open}
            setOpen={setOpen}
            onSuccess={() => {
                setOpen(false);
            }}
            onCancel={() => {
                setOpen(false);
            }}
            onAccept={() => {
                handleSubmit();
            }}
            aria-labelledby="modal-title"
            aria-describedby="modal-description"
        >
            <Stack spacing={2}>
                <Heading>{`Delete word filter (#${record.word_id})?`}</Heading>
            </Stack>
        </ConfirmationModal>
    );
};
