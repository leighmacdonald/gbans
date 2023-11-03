import React, { useCallback, useState } from 'react';
import NiceModal from '@ebay/nice-modal-react';
import FormControl from '@mui/material/FormControl';
import Stack from '@mui/material/Stack';
import TextField from '@mui/material/TextField';
import { apiDeleteBan } from '../../api';
import { useUserFlashCtx } from '../../contexts/UserFlashCtx';
import { Heading } from '../Heading';
import { ConfirmationModal, ConfirmationModalProps } from './ConfirmationModal';

export interface UnbanModalProps<Ban> extends ConfirmationModalProps<Ban> {
    banId: number;
    personaName: string;
}

export const UnbanSteamModal = NiceModal.create(
    ({ onSuccess, banId, personaName }: UnbanModalProps<null>) => {
        const [reasonText, setReasonText] = useState<string>('');

        const { sendFlash } = useUserFlashCtx();

        const handleSubmit = useCallback(() => {
            if (reasonText == '') {
                sendFlash('error', 'Reason cannot be empty');
                return;
            }
            apiDeleteBan(banId, reasonText)
                .then(() => {
                    sendFlash('success', `Unbanned successfully`);
                    onSuccess && onSuccess(null);
                })
                .catch((err) => {
                    sendFlash('error', `Failed to unban: ${err}`);
                });
        }, [reasonText, banId, sendFlash, onSuccess]);

        return (
            <ConfirmationModal
                onAccept={() => {
                    handleSubmit();
                }}
                aria-labelledby="modal-title"
                aria-describedby="modal-description"
                id={'confirm-unban-steam'}
            >
                <Stack spacing={2}>
                    <Heading>{`Unban Player: ${personaName}`}</Heading>
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
    }
);
