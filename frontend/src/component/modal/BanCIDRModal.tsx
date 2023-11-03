import React, { useCallback } from 'react';
import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import GavelIcon from '@mui/icons-material/Gavel';
import {
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle
} from '@mui/material';
import Stack from '@mui/material/Stack';
import { Formik } from 'formik';
import * as yup from 'yup';
import { apiCreateBanCIDR, BanReason, BanType, Duration } from '../../api';
import { useUserFlashCtx } from '../../contexts/UserFlashCtx';
import { logErr } from '../../util/errors';
import { Heading } from '../Heading';
import {
    BanReasonField,
    BanReasonFieldValidator
} from '../formik/BanReasonField';
import {
    BanReasonTextField,
    BanReasonTextFieldValidator
} from '../formik/BanReasonTextField';
import { BanTypeField, BanTypeFieldValidator } from '../formik/BanTypeField';
import {
    DurationCustomField,
    DurationCustomFieldValidator
} from '../formik/DurationCustomField';
import { DurationField, DurationFieldValidator } from '../formik/DurationField';
import {
    NetworkRangeField,
    NetworkRangeFieldValidator
} from '../formik/NetworkRangeField';
import { NoteField, NoteFieldValidator } from '../formik/NoteField';
import {
    SteamIdField,
    SteamIDInputValue,
    steamIdValidator
} from '../formik/SteamIdField';
import { CancelButton, ResetButton, SaveButton } from './Buttons';

interface BanCIDRFormValues extends SteamIDInputValue {
    cidr: string;
    ban_type: BanType;
    reason: BanReason;
    reason_text: string;
    duration: Duration;
    duration_custom: string;
    note: string;
}

export const validationSchema = yup.object({
    steam_id: steamIdValidator,
    cidr: NetworkRangeFieldValidator,
    banType: BanTypeFieldValidator,
    reason: BanReasonFieldValidator,
    reasonText: BanReasonTextFieldValidator,
    duration: DurationFieldValidator,
    durationCustom: DurationCustomFieldValidator,
    note: NoteFieldValidator
});

export const BanCIDRModal = NiceModal.create(() => {
    const { sendFlash } = useUserFlashCtx();
    const modal = useModal();
    const onSubmit = useCallback(
        async (values: BanCIDRFormValues) => {
            try {
                await apiCreateBanCIDR({
                    note: values.note,
                    ban_type: values.ban_type,
                    duration: values.duration,
                    reason: values.reason,
                    reason_text: values.reason_text,
                    target_id: values.steam_id,
                    cidr: values.cidr
                });
                sendFlash('success', 'Ban created successfully');
            } catch (e) {
                logErr(e);
                sendFlash('error', 'Error saving ban');
            } finally {
                await modal.hide();
            }
        },
        [modal, sendFlash]
    );

    const formId = 'banCIDRForm';

    return (
        <Formik
            onSubmit={onSubmit}
            id={formId}
            initialValues={{
                ban_type: BanType.NoComm,
                duration: Duration.dur2w,
                duration_custom: '',
                note: '',
                reason: BanReason.Cheating,
                steam_id: '',
                reason_text: '',
                cidr: ''
            }}
            validateOnBlur={true}
            validateOnChange={false}
            //validationSchema={validationSchema}
        >
            <Dialog fullWidth {...muiDialogV5(modal)}>
                <DialogTitle component={Heading} iconLeft={<GavelIcon />}>
                    Ban CIDR Range
                </DialogTitle>

                <DialogContent>
                    <Stack spacing={2}>
                        <SteamIdField fullWidth />
                        <NetworkRangeField />
                        <BanTypeField />
                        <BanReasonField />
                        <BanReasonTextField />
                        <DurationField />
                        <DurationCustomField />
                        <NoteField />
                    </Stack>
                </DialogContent>
                <DialogActions>
                    <CancelButton />
                    <ResetButton />
                    <SaveButton />
                </DialogActions>
            </Dialog>
        </Formik>
    );
});
