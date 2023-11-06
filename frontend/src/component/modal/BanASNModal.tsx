import React, { useCallback } from 'react';
import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import LanIcon from '@mui/icons-material/Lan';
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
    apiCreateBanASN,
    apiUpdateBanASN,
    BanReason,
    Duration,
    IAPIBanASNRecord
} from '../../api';
import { Heading } from '../Heading';
import { ASNumberField, ASNumberFieldValidator } from '../formik/ASNumberField';
import {
    BanReasonField,
    BanReasonFieldValidator
} from '../formik/BanReasonField';
import {
    BanReasonTextField,
    BanReasonTextFieldValidator
} from '../formik/BanReasonTextField';
import {
    DurationCustomField,
    DurationCustomFieldValidator
} from '../formik/DurationCustomField';
import { DurationField, DurationFieldValidator } from '../formik/DurationField';
import { NoteField, NoteFieldValidator } from '../formik/NoteField';
import {
    SteamIdField,
    SteamIDInputValue,
    steamIdValidator
} from '../formik/SteamIdField';
import { CancelButton, ResetButton, SubmitButton } from './Buttons';

interface BanASNFormValues extends SteamIDInputValue {
    ban_asn_id?: number;
    as_num: number;
    reason: BanReason;
    reason_text: string;
    duration: Duration;
    duration_custom: Date;
    note: string;
}

export const validationSchema = yup.object({
    steam_id: steamIdValidator,
    as_num: ASNumberFieldValidator,
    reason: BanReasonFieldValidator,
    reason_text: BanReasonTextFieldValidator,
    duration: DurationFieldValidator,
    duration_custom: DurationCustomFieldValidator,
    note: NoteFieldValidator
});

export interface BanASNModalProps {
    existing?: IAPIBanASNRecord;
}

export const BanASNModal = NiceModal.create(
    ({ existing }: BanASNModalProps) => {
        const modal = useModal();
        const onSubmit = useCallback(
            async (values: BanASNFormValues) => {
                try {
                    if (existing && existing.as_num > 0) {
                        modal.resolve(
                            await apiUpdateBanASN(existing.as_num, {
                                note: values.note,
                                valid_until: values.duration_custom,
                                reason: values.reason,
                                reason_text: values.reason_text,
                                target_id: values.steam_id
                            })
                        );
                    } else {
                        modal.resolve(
                            await apiCreateBanASN({
                                note: values.note,
                                duration: values.duration,
                                valid_until: values.duration_custom,
                                reason: values.reason,
                                reason_text: values.reason_text,
                                target_id: values.steam_id,
                                as_num: values.as_num
                            })
                        );
                    }
                    await modal.hide();
                } catch (e) {
                    modal.resolve(e);
                }
            },
            [existing, modal]
        );

        const formId = 'banASNForm';

        return (
            <Formik
                onSubmit={onSubmit}
                id={formId}
                initialValues={{
                    ban_asn_id: existing?.ban_asn_id,
                    duration: existing ? Duration.durCustom : Duration.dur2w,
                    duration_custom: existing
                        ? existing.valid_until
                        : new Date(),
                    note: existing ? existing.note : '',
                    reason: existing ? existing.reason : BanReason.Cheating,
                    steam_id: existing ? existing.target_id : '',
                    reason_text: existing ? existing.reason_text : '',
                    as_num: existing ? existing.as_num : 0
                }}
                validateOnBlur={true}
                validateOnChange={false}
                validationSchema={validationSchema}
            >
                <Dialog fullWidth {...muiDialogV5(modal)}>
                    <DialogTitle component={Heading} iconLeft={<LanIcon />}>
                        Ban Autonomous System Number Range
                    </DialogTitle>

                    <DialogContent>
                        <Stack spacing={2}>
                            <SteamIdField fullWidth isReadOnly={false} />
                            <ASNumberField />
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
