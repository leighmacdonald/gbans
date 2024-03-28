import { useCallback, useState } from 'react';
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
    ASNBanRecord,
    APIError
} from '../../api';
import {
    asNumberFieldValidator,
    banReasonFieldValidator,
    banReasonTextFieldValidator,
    steamIdValidator
} from '../../util/validators.ts';
import { Heading } from '../Heading';
import { ASNumberField } from '../formik/ASNumberField';
import { BanReasonField } from '../formik/BanReasonField';
import { BanReasonTextField } from '../formik/BanReasonTextField';
import {
    DurationCustomField,
    DurationCustomFieldValidator
} from '../formik/DurationCustomField';
import { DurationField, DurationFieldValidator } from '../formik/DurationField';
import { ErrorField } from '../formik/ErrorField';
import { NoteField, NoteFieldValidator } from '../formik/NoteField';
import { TargetIDField, TargetIDInputValue } from '../formik/TargetIdField.tsx';
import { CancelButton, ResetButton, SubmitButton } from './Buttons';

type BanASNFormValues = {
    ban_asn_id?: number;
    as_num: number;
    reason: BanReason;
    reason_text: string;
    duration: Duration;
    duration_custom: Date;
    note: string;
} & TargetIDInputValue;

const validationSchema = yup.object({
    target_id: steamIdValidator('target_id'),
    as_num: asNumberFieldValidator,
    reason: banReasonFieldValidator,
    reason_text: banReasonTextFieldValidator,
    duration: DurationFieldValidator,
    duration_custom: DurationCustomFieldValidator,
    note: NoteFieldValidator
});

export interface BanASNModalProps {
    existing?: ASNBanRecord;
}

export const BanASNModal = NiceModal.create(
    ({ existing }: BanASNModalProps) => {
        const [error, setError] = useState<string>();
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
                                target_id: values.target_id
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
                                target_id: values.target_id,
                                as_num: values.as_num
                            })
                        );
                    }
                    await modal.hide();
                    setError(undefined);
                } catch (e) {
                    modal.resolve(e);
                    if (e instanceof APIError) {
                        setError(e.message);
                    } else {
                        setError('Unknown internal error');
                    }
                }
            },
            [existing, modal]
        );

        return (
            <Formik
                onSubmit={onSubmit}
                id={'banASNForm'}
                initialValues={{
                    ban_asn_id: existing?.ban_asn_id,
                    duration: existing ? Duration.durCustom : Duration.dur2w,
                    duration_custom: existing
                        ? existing.valid_until
                        : new Date(),
                    note: existing ? existing.note : '',
                    reason: existing ? existing.reason : BanReason.Cheating,
                    target_id: existing ? existing.target_id : '',
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
                            <TargetIDField />
                            <ASNumberField />
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

export default BanASNModal;
