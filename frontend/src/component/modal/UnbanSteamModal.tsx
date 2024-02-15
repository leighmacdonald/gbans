import { useCallback } from 'react';
import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import {
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle
} from '@mui/material';
import Stack from '@mui/material/Stack';
import { Formik } from 'formik';
import { apiDeleteBan } from '../../api';
import {
    UnbanReasonTextField,
    unbanValidationSchema
} from '../formik/UnbanReasonTextField';
import { CancelButton, SubmitButton } from './Buttons';

export interface UnbanModalProps {
    banId: number; // common placeholder for any primary key id for a ban
    personaName?: string;
}

export interface UnbanFormValues {
    unban_reason: string;
}

export const UnbanSteamModal = NiceModal.create(
    ({ banId, personaName }: UnbanModalProps) => {
        const modal = useModal();

        const onSubmit = useCallback(
            async (values: UnbanFormValues) => {
                if (values.unban_reason == '') {
                    modal.reject({ error: 'Reason cannot be empty' });
                    return;
                }
                try {
                    await apiDeleteBan(banId, values.unban_reason);
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
            <Formik
                initialValues={{ unban_reason: '' }}
                onSubmit={onSubmit}
                validateOnChange={true}
                validationSchema={unbanValidationSchema}
            >
                <Dialog {...muiDialogV5(modal)}>
                    <DialogTitle>
                        Unban {personaName} (#{banId})
                    </DialogTitle>

                    <DialogContent>
                        <Stack spacing={2}>
                            <UnbanReasonTextField />
                        </Stack>
                    </DialogContent>

                    <DialogActions>
                        <CancelButton />
                        <SubmitButton />
                    </DialogActions>
                </Dialog>
            </Formik>
        );
    }
);

export default UnbanSteamModal;
