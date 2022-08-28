import React, { useCallback, useState } from 'react';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import { apiDeleteGroupBan, IAPIBanGroupRecord } from '../api';
import { ConfirmationModal, ConfirmationModalProps } from './ConfirmationModal';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { Heading } from './Heading';

export interface UnbanGroupModalProps
    extends ConfirmationModalProps<IAPIBanGroupRecord> {
    record: IAPIBanGroupRecord;
}

export const UnbanGroupModal = ({
    open,
    setOpen,
    onSuccess,
    record
}: UnbanGroupModalProps) => {
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
};
