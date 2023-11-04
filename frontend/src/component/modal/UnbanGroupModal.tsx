import React, { useCallback } from 'react';
import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import {
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle
} from '@mui/material';
import Stack from '@mui/material/Stack';
import { Formik } from 'formik';
import { apiDeleteGroupBan } from '../../api';
import { BanReasonTextField } from '../formik/BanReasonTextField';
import { CancelButton, SaveButton } from './Buttons';
import { UnbanFormValues, UnbanModalProps } from './UnbanSteamModal';

export const UnbanGroupModal = NiceModal.create(
    ({ banId }: UnbanModalProps) => {
        const modal = useModal();

        const onSubmit = useCallback(
            async (values: UnbanFormValues) => {
                if (values.reason_text == '') {
                    modal.reject({ error: 'Reason cannot be empty' });
                    await modal.hide();
                    return;
                }
                try {
                    await apiDeleteGroupBan(banId, values.reason_text);
                    modal.resolve();
                } catch (e) {
                    modal.reject(e);
                } finally {
                    await modal.hide();
                }
            },
            [banId, modal]
        );

        return (
            <Formik<UnbanFormValues>
                initialValues={{ reason_text: '' }}
                onSubmit={onSubmit}
            >
                <Dialog {...muiDialogV5(modal)}>
                    <DialogTitle>Unban Steam Group (#{banId})</DialogTitle>

                    <DialogContent>
                        <Stack spacing={2}>
                            <BanReasonTextField paired={false} />
                        </Stack>
                    </DialogContent>

                    <DialogActions>
                        <CancelButton />
                        <SaveButton />
                    </DialogActions>
                </Dialog>
            </Formik>
        );
    }
);
