import React, { useCallback, useState } from 'react';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import { apiDeleteASNBan, IAPIBanASNRecord } from '../api';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { ConfirmationModal, ConfirmationModalProps } from './ConfirmationModal';
import { Heading } from './Heading';

export interface UnbanASNModalProps
    extends ConfirmationModalProps<IAPIBanASNRecord> {
    record: IAPIBanASNRecord;
}

export const UnbanASNModal = ({
    open,
    setOpen,
    onSuccess,
    record
}: UnbanASNModalProps) => {
    const [reasonText, setReasonText] = useState<string>('');
    const { sendFlash } = useUserFlashCtx();

    const handleSubmit = useCallback(() => {
        if (reasonText == '') {
            sendFlash('error', 'Reason cannot be empty');
            return;
        }
        apiDeleteASNBan(record.as_num, reasonText)
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
                        Unban ASN (#{record.ban_asn_id}): {record.as_num}
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
