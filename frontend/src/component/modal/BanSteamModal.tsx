import React, { useCallback, useMemo } from 'react';
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
import { apiCreateBanSteam, BanReason, BanType, Duration } from '../../api';
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
import { IncludeFriendsField } from '../formik/IncludeFriendsField';
import { NoteField, NoteFieldValidator } from '../formik/NoteField';
import { ReportIdField, ReportIdFieldValidator } from '../formik/ReportIdField';
import {
    SteamIdField,
    SteamIDInputValue,
    steamIdValidator
} from '../formik/SteamIdField';
import { CancelButton, ResetButton, SaveButton } from './Buttons';

export interface BanModalProps {
    open: boolean;
    setOpen: (open: boolean) => void;
    reportId?: number;
    steamId?: string;
}

interface BanSteamFormValues extends SteamIDInputValue {
    report_id?: number;
    ban_type: BanType;
    reason: BanReason;
    reason_text: string;
    duration: Duration;
    duration_custom: string;
    note: string;
    include_friends: boolean;
}

export const validationSchema = yup.object({
    steam_id: steamIdValidator,
    reportId: ReportIdFieldValidator,
    banType: BanTypeFieldValidator,
    reason: BanReasonFieldValidator,
    reasonText: BanReasonTextFieldValidator,
    duration: DurationFieldValidator,
    durationCustom: DurationCustomFieldValidator,
    note: NoteFieldValidator
});

export const BanSteamModal = NiceModal.create(
    ({ steamId, reportId }: BanModalProps) => {
        const { sendFlash } = useUserFlashCtx();
        const modal = useModal();
        const isReadOnlySid = useMemo(() => {
            return !!steamId;
        }, [steamId]);
        const onSumit = useCallback(
            async (values: BanSteamFormValues) => {
                try {
                    await apiCreateBanSteam({
                        note: values.note,
                        ban_type: values.ban_type,
                        duration: values.duration,
                        reason: values.reason,
                        reason_text: values.reason_text,
                        report_id: values.report_id,
                        target_id: values.steam_id,
                        include_friends: values.include_friends
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

        const formId = 'banSteamForm';

        return (
            <Formik
                onSubmit={onSumit}
                id={formId}
                initialValues={{
                    ban_type: BanType.NoComm,
                    duration: Duration.dur2w,
                    duration_custom: '',
                    note: '',
                    reason: BanReason.Cheating,
                    steam_id: steamId ?? '',
                    reason_text: '',
                    report_id: reportId,
                    include_friends: false
                }}
                validateOnBlur={true}
                validateOnChange={true}
                //validationSchema={validationSchema}
            >
                <Dialog fullWidth {...muiDialogV5(modal)}>
                    <DialogTitle component={Heading} iconLeft={<GavelIcon />}>
                        Ban Steam Profile
                    </DialogTitle>

                    <DialogContent>
                        <Stack spacing={2}>
                            <SteamIdField
                                fullWidth
                                isReadOnly={isReadOnlySid}
                            />
                            <ReportIdField />
                            <BanTypeField />
                            <BanReasonField />
                            <BanReasonTextField />
                            <IncludeFriendsField />
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
    }
);
