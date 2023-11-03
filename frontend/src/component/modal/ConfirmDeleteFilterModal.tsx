import React, { useCallback } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import Stack from '@mui/material/Stack';
import { apiDeleteFilter, Filter } from '../../api/filters';
import { useUserFlashCtx } from '../../contexts/UserFlashCtx';
import { logErr } from '../../util/errors';
import { Heading } from '../Heading';
import { ConfirmationModal, ConfirmationModalProps } from './ConfirmationModal';

export interface ConfirmDeleteFilterModalProps
    extends ConfirmationModalProps<Filter> {
    record: Filter;
}

export const ConfirmDeleteFilterModal = NiceModal.create(
    ({ onSuccess, record }: ConfirmDeleteFilterModalProps) => {
        const { sendFlash } = useUserFlashCtx();

        const handleSubmit = useCallback(() => {
            if (!record.filter_id) {
                logErr(new Error('filter_id not present, cannot delete'));
                return;
            }
            apiDeleteFilter(record.filter_id)
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
                id={'modal-confirm-delete-filter'}
                onAccept={() => {
                    handleSubmit();
                }}
                aria-labelledby="modal-title"
                aria-describedby="modal-description"
            >
                <Stack spacing={2}>
                    <Heading>{`Delete word filter (#${record.filter_id})?`}</Heading>
                </Stack>
            </ConfirmationModal>
        );
    }
);
