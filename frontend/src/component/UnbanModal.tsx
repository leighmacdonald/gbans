import React, { useCallback, useState } from 'react';
import Stack from '@mui/material/Stack';
import { apiDeleteBan, IAPIBanRecordProfile } from '../api';
import { ConfirmationModal, ConfirmationModalProps } from './ConfirmationModal';
import FormControl from '@mui/material/FormControl';
import TextField from '@mui/material/TextField';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { Heading } from './Heading';

export interface UnbanModalProps<Ban> extends ConfirmationModalProps<Ban> {
    banRecord: IAPIBanRecordProfile;
}

export const UnbanModal = ({
    open,
    setOpen,
    onSuccess,
    banRecord
}: UnbanModalProps<null>) => {
    const [bp] = useState<IAPIBanRecordProfile>(banRecord);
    const [reasonText, setReasonText] = useState<string>('');

    const { sendFlash } = useUserFlashCtx();

    const handleSubmit = useCallback(() => {
        if (reasonText == '') {
            sendFlash('error', 'Reason cannot be empty');
            return;
        }
        apiDeleteBan(bp.ban_id, reasonText)
            .then((resp) => {
                sendFlash('success', `Unbanned successfully`);
                onSuccess && onSuccess(resp);
            })
            .catch((err) => {
                sendFlash('error', `Failed to unban: ${err}`);
            });
    }, [reasonText, bp.ban_id, sendFlash, onSuccess]);

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
                        Unban Player:
                        {bp.personaname || `${bp.steam_id}`}
                    </>
                </Heading>
                <Stack spacing={3} alignItems={'center'}>
                    <FormControl fullWidth>
                        <TextField
                            label={'Reason'}
                            id={'reasonText'}
                            value={reasonText}
                            onChange={(evt) => {
                                setReasonText(evt.target.value);
                            }}
                        />
                    </FormControl>
                </Stack>
            </Stack>
        </ConfirmationModal>
    );
};
