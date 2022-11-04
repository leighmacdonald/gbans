import React, { useCallback, useState } from 'react';
import Stack from '@mui/material/Stack';
import FormControl from '@mui/material/FormControl';
import TextField from '@mui/material/TextField';
import { apiDeleteBan } from '../api';
import { ConfirmationModal, ConfirmationModalProps } from './ConfirmationModal';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { Heading } from './Heading';

export interface UnbanModalProps<Ban> extends ConfirmationModalProps<Ban> {
    banId: number;
    personaName: string;
}

export const UnbanSteamModal = ({
    open,
    setOpen,
    onSuccess,
    banId,
    personaName
}: UnbanModalProps<null>) => {
    const [reasonText, setReasonText] = useState<string>('');

    const { sendFlash } = useUserFlashCtx();

    const handleSubmit = useCallback(() => {
        if (reasonText == '') {
            sendFlash('error', 'Reason cannot be empty');
            return;
        }
        apiDeleteBan(banId, reasonText)
            .then((resp) => {
                if (!resp.status) {
                    sendFlash('error', `Failed to unban`);
                    return;
                }
                sendFlash('success', `Unbanned successfully`);
                onSuccess && onSuccess(null);
            })
            .catch((err) => {
                sendFlash('error', `Failed to unban: ${err}`);
            });
    }, [reasonText, banId, sendFlash, onSuccess]);

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
                setOpen(false);
            }}
            aria-labelledby="modal-title"
            aria-describedby="modal-description"
        >
            <Stack spacing={2}>
                <Heading>
                    <>Unban Player: {personaName}</>
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
