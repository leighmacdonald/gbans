import { useCallback, useMemo, useState } from 'react';
import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import DirectionsRunIcon from '@mui/icons-material/DirectionsRun';
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
    apiCreateBanSteam,
    APIError,
    apiUpdateBanSteam,
    BanReason,
    BanType,
    Duration,
    SteamBanRecord
} from '../../api';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import {
    banReasonFieldValidator,
    banReasonTextFieldValidator,
    steamIdValidator
} from '../../util/validators.ts';
import { Heading } from '../Heading';
import { BanReasonField } from '../formik/BanReasonField';
import { BanReasonTextField } from '../formik/BanReasonTextField';
import { BanTypeField, BanTypeFieldValidator } from '../formik/BanTypeField';
import {
    DurationCustomField,
    DurationCustomFieldValidator
} from '../formik/DurationCustomField';
import { DurationField, DurationFieldValidator } from '../formik/DurationField';
import { ErrorField } from '../formik/ErrorField';
import { IncludeFriendsField } from '../formik/IncludeFriendsField';
import { NoteField, NoteFieldValidator } from '../formik/NoteField';
import { ReportIdField, ReportIdFieldValidator } from '../formik/ReportIdField';
import { TargetIDField, TargetIDInputValue } from '../formik/TargetIdField.tsx';
import { CancelButton, ResetButton, SubmitButton } from './Buttons';

export interface BanModalProps {
    reportId?: number;
    steamId?: string;
    existing?: SteamBanRecord;
}

type BanSteamFormValues = {
    report_id?: number;
    ban_type: BanType;
    reason: BanReason;
    reason_text: string;
    duration: Duration;
    duration_custom: Date;
    note: string;
    include_friends: boolean;
    existing?: SteamBanRecord;
} & TargetIDInputValue;

const validationSchema = yup.object({
    target_id: steamIdValidator('target_id'),
    reportId: ReportIdFieldValidator,
    ban_type: BanTypeFieldValidator,
    reason: banReasonFieldValidator,
    reason_text: banReasonTextFieldValidator,
    duration: DurationFieldValidator,
    duration_custom: DurationCustomFieldValidator,
    note: NoteFieldValidator
});

export const BanSteamModal = NiceModal.create(
    ({ steamId, reportId, existing }: BanModalProps) => {
        const [error, setError] = useState<string>();
        const { sendFlash } = useUserFlashCtx();
        const modal = useModal();

        const isReadOnlySid = useMemo(() => {
            return !!steamId || (existing && existing?.ban_id > 0);
        }, [existing, steamId]);

        const isUpdate = useMemo(() => {
            return existing && existing?.ban_id > 0;
        }, [existing]);

        const onSubmit = useCallback(
            async (values: BanSteamFormValues) => {
                try {
                    if (isUpdate && existing) {
                        const ban_record = await apiUpdateBanSteam(
                            existing.ban_id,
                            {
                                note: values.note,
                                ban_type: values.ban_type,
                                reason: values.reason,
                                reason_text: values.reason_text,
                                include_friends: values.include_friends,
                                valid_until: values.duration_custom
                            }
                        );
                        sendFlash('success', 'Updated ban successfully');
                        modal.resolve(ban_record);
                    } else {
                        const ban_record = await apiCreateBanSteam({
                            note: values.note,
                            ban_type: values.ban_type,
                            duration: values.duration,
                            valid_until: values.duration_custom,
                            reason: values.reason,
                            reason_text: values.reason_text,
                            report_id: values.report_id,
                            target_id: values.target_id,
                            include_friends: values.include_friends
                        });
                        sendFlash('success', 'Created ban successfully');
                        modal.resolve(ban_record);
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
            [existing, isUpdate, modal, sendFlash]
        );

        return (
            <Formik
                onSubmit={onSubmit}
                id={'banSteamForm'}
                initialValues={{
                    ban_type: existing?.ban_type ?? BanType.Banned,
                    duration:
                        existing?.ban_id && existing?.ban_id > 0
                            ? Duration.durCustom
                            : Duration.dur2w,
                    duration_custom:
                        existing?.ban_id && existing?.ban_id > 0
                            ? existing?.valid_until
                            : new Date(),
                    note: existing?.note ?? '',
                    reason: existing?.reason ?? BanReason.Cheating,
                    target_id: existing?.target_id ?? steamId ?? '',
                    reason_text: existing?.reason_text ?? '',
                    report_id: existing?.report_id ?? reportId,
                    include_friends: existing?.include_friends ?? false,
                    existing: existing
                }}
                validateOnBlur={true}
                validateOnChange={false}
                validationSchema={validationSchema}
            >
                <Dialog fullWidth {...muiDialogV5(modal)}>
                    <DialogTitle
                        component={Heading}
                        iconLeft={<DirectionsRunIcon />}
                    >
                        Ban Steam Profile
                    </DialogTitle>

                    <DialogContent>
                        <Stack spacing={2}>
                            <TargetIDField isReadOnly={isReadOnlySid} />
                            <ReportIdField />
                            <BanTypeField />
                            <BanReasonField />
                            <BanReasonTextField paired={true} />
                            <IncludeFriendsField />
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

export default BanSteamModal;
