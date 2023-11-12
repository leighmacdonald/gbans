import React, { useCallback } from 'react';
import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import RouterIcon from '@mui/icons-material/Router';
import {
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle
} from '@mui/material';
import Stack from '@mui/material/Stack';
import { Formik } from 'formik';
import * as yup from 'yup';
import {
    apiCreateBanCIDR,
    apiUpdateBanCIDR,
    BanReason,
    BanType,
    Duration,
    CIDRBanRecord
} from '../../api';
import { useUserFlashCtx } from '../../contexts/UserFlashCtx';
import { logErr } from '../../util/errors';
import { Heading } from '../Heading';
import {
    BanReasonField,
    BanReasonFieldValidator
} from '../formik/BanReasonField';
import {
    BanReasonTextField,
    banReasonTextFieldValidator
} from '../formik/BanReasonTextField';
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
import { CancelButton, ResetButton, SubmitButton } from './Buttons';

interface BanCIDRFormValues extends SteamIDInputValue {
    cidr: string;
    reason: BanReason;
    reason_text: string;
    duration: Duration;
    duration_custom: Date;
    note: string;
    existing?: CIDRBanRecord;
}

const validationSchema = yup.object({
    steam_id: steamIdValidator,
    cidr: NetworkRangeFieldValidator,
    reason: BanReasonFieldValidator,
    reason_text: banReasonTextFieldValidator,
    duration: DurationFieldValidator,
    duration_custom: DurationCustomFieldValidator,
    note: NoteFieldValidator
});

export interface BanCIDRModalProps {
    existing?: CIDRBanRecord;
}

export const BanCIDRModal = NiceModal.create(
    ({ existing }: BanCIDRModalProps) => {
        const { sendFlash } = useUserFlashCtx();
        const modal = useModal();

        const onSubmit = useCallback(
            async (values: BanCIDRFormValues) => {
                try {
                    if (existing && existing?.net_id > 0) {
                        modal.resolve(
                            await apiUpdateBanCIDR(existing?.net_id, {
                                note: values.note,
                                reason: values.reason,
                                valid_until: values.duration_custom,
                                reason_text: values.reason_text,
                                target_id: values.steam_id,
                                cidr: values.cidr
                            })
                        );
                    } else {
                        modal.resolve(
                            await apiCreateBanCIDR({
                                note: values.note,
                                duration: values.duration,
                                valid_until: values.duration_custom,
                                reason: values.reason,
                                reason_text: values.reason_text,
                                target_id: values.steam_id,
                                cidr: values.cidr
                            })
                        );
                    }
                    await modal.hide();
                } catch (e) {
                    logErr(e);
                    sendFlash('error', 'Error saving ban');
                }
            },
            [existing, modal, sendFlash]
        );

        const formId = 'banCIDRForm';

        return (
            <Formik
                onSubmit={onSubmit}
                id={formId}
                initialValues={{
                    ban_type: existing ? existing.ban_type : BanType.Banned,
                    duration: existing ? Duration.durCustom : Duration.durInf,
                    duration_custom: existing
                        ? existing.valid_until
                        : new Date(),
                    note: existing ? existing.note : '',
                    reason: existing ? existing.reason : BanReason.Cheating,
                    steam_id: existing ? existing.target_id : '',
                    reason_text: existing ? existing.reason_text : '',
                    cidr: existing ? existing.cidr : ''
                }}
                validateOnBlur={true}
                validateOnChange={false}
                validationSchema={validationSchema}
            >
                <Dialog fullWidth {...muiDialogV5(modal)}>
                    <DialogTitle component={Heading} iconLeft={<RouterIcon />}>
                        Ban CIDR Range
                    </DialogTitle>

                    <DialogContent>
                        <Stack spacing={2}>
                            <SteamIdField />
                            <NetworkRangeField />
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
                        <SubmitButton />
                    </DialogActions>
                </Dialog>
            </Formik>
        );
    }
);
