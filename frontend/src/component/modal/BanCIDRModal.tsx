import React, { useCallback, useState } from 'react';
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
    CIDRBanRecord,
    APIError
} from '../../api';
import { Heading } from '../Heading';
import {
    BanReasonField,
    banReasonFieldValidator
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
import { ErrorField } from '../formik/ErrorField';
import {
    makeNetworkRangeFieldValidator,
    NetworkRangeField
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
    cidr: makeNetworkRangeFieldValidator(true),
    reason: banReasonFieldValidator,
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
        const [error, setError] = useState<string>();
        const modal = useModal();

        const onSubmit = useCallback(
            async (values: BanCIDRFormValues) => {
                try {
                    const realCidr = values.cidr.includes('/')
                        ? values.cidr
                        : `${values.cidr}/32`;
                    if (existing && existing?.net_id > 0) {
                        modal.resolve(
                            await apiUpdateBanCIDR(existing?.net_id, {
                                note: values.note,
                                reason: values.reason,
                                valid_until: values.duration_custom,
                                reason_text: values.reason_text,
                                target_id: values.steam_id,
                                cidr: realCidr
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
                                cidr: realCidr
                            })
                        );
                    }
                    await modal.hide();
                    setError(undefined);
                } catch (e) {
                    modal.reject(e);
                    if (e instanceof APIError) {
                        setError(e.message);
                    } else {
                        setError('Unknown internal error');
                    }
                }
            },
            [existing, modal]
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
                            <ErrorField error={error} />
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
