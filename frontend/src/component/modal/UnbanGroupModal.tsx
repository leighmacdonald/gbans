import React, { useCallback, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import { apiDeleteGroupBan, IAPIBanGroupRecord } from '../../api';
import { useUserFlashCtx } from '../../contexts/UserFlashCtx';
import { Heading } from '../Heading';
import { ConfirmationModal, ConfirmationModalProps } from './ConfirmationModal';

export interface UnbanGroupModalProps
    extends ConfirmationModalProps<IAPIBanGroupRecord> {
    record: IAPIBanGroupRecord;
}

export const UnbanGroupModal = NiceModal.create(
    ({ onSuccess, record }: UnbanGroupModalProps) => {
        const [reasonText, setReasonText] = useState<string>('');
        const { sendFlash } = useUserFlashCtx();

        const handleSubmit = useCallback(() => {
            if (reasonText == '') {
                sendFlash('error', 'Reason cannot be empty');
                return;
            }
            apiDeleteGroupBan(record.ban_group_id, reasonText)
                .then(() => {
                    sendFlash('success', `Unbanned successfully`);
                    onSuccess && onSuccess(record);
                })
                .catch((err) => {
                    sendFlash('error', `Failed to unban: ${err}`);
                });
        }, [reasonText, record, sendFlash, onSuccess]);

        return (
            <ConfirmationModal
                id={'modal-unban-group'}
                onAccept={() => {
                    handleSubmit();
                }}
                aria-labelledby="modal-title"
                aria-describedby="modal-description"
            >
                <Stack spacing={2}>
                    <Heading>
                        <>
                            Unban Group (#{record.ban_group_id}):
                            {record.group_id.toString()}
                        </>
                    </Heading>
                    <Stack spacing={3} alignItems={'center'}>
                        <TextField
                            fullWidth
                            label={'Reason'}
                            id={'reasonText'}
                            value={reasonText}
                            onChange={(evt) => {
                                setReasonText(evt.target.value);
                            }}
                        />
                    </Stack>
                </Stack>
            </ConfirmationModal>
        );
    }
);
