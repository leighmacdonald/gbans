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
import { apiDeleteCIDRBan } from '../../api';
import {
    BanReasonTextField,
    unbanValidationSchema
} from '../formik/BanReasonTextField';
import { CancelButton, SubmitButton } from './Buttons';
import { UnbanFormValues, UnbanModalProps } from './UnbanSteamModal';

export const UnbanCIDRModal = NiceModal.create(({ banId }: UnbanModalProps) => {
    const modal = useModal();

    const onSubmit = useCallback(
        async (values: UnbanFormValues) => {
            if (values.reason_text == '') {
                modal.reject({ error: 'Reason cannot be empty' });
                await modal.hide();
                return;
            }
            try {
                await apiDeleteCIDRBan(banId, values.reason_text);
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
            validateOnChange={true}
            validationSchema={unbanValidationSchema}
        >
            <Dialog {...muiDialogV5(modal)}>
                <DialogTitle>Unban CIDR (#{banId})</DialogTitle>

                <DialogContent>
                    <Stack spacing={2}>
                        <BanReasonTextField paired={false} />
                    </Stack>
                </DialogContent>

                <DialogActions>
                    <CancelButton />
                    <SubmitButton />
                </DialogActions>
            </Dialog>
        </Formik>
    );
});
