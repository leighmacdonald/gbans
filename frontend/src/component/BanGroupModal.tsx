import React from 'react';
import Stack from '@mui/material/Stack';
import {
    apiCreateBanGroup,
    BanReason,
    BanType,
    Duration,
    IAPIBanGroupRecord
} from '../api';
import { ConfirmationModalProps } from './ConfirmationModal';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { Heading } from './Heading';
import { logErr } from '../util/errors';
import { useFormik } from 'formik';
import * as yup from 'yup';
import { Dialog, DialogContent, DialogTitle } from '@mui/material';
import GavelIcon from '@mui/icons-material/Gavel';
import { BanTypeField, BanTypeFieldValidator } from './formik/BanTypeField';
import {
    BanReasonField,
    BanReasonFieldValidator
} from './formik/BanReasonField';
import { DurationField, DurationFieldValidator } from './formik/DurationField';
import {
    DurationCustomField,
    DurationCustomFieldValidator
} from './formik/DurationCustomField';
import { NoteField, NoteFieldValidator } from './formik/NoteField';
import { ModalButtons } from './formik/ModalButtons';
import { GroupIdField, GroupIdFieldValidator } from './formik/GroupIdField';
import {
    BanReasonTextField,
    BanReasonTextFieldValidator
} from './formik/BanReasonTextField';

export interface BanGroupModalProps
    extends ConfirmationModalProps<IAPIBanGroupRecord> {
    asnNum?: number;
}

interface BanGroupFormValues {
    groupId: string;
    banType: BanType;
    reason: BanReason;
    reasonText: string;
    duration: Duration;
    durationCustom: string;
    note: string;
}

const validationSchema = yup.object({
    groupId: GroupIdFieldValidator,
    banType: BanTypeFieldValidator,
    reason: BanReasonFieldValidator,
    reasonText: BanReasonTextFieldValidator,
    duration: DurationFieldValidator,
    durationCustom: DurationCustomFieldValidator,
    note: NoteFieldValidator
});

export const BanGroupModal = ({ open, setOpen }: BanGroupModalProps) => {
    const { sendFlash } = useUserFlashCtx();

    const formik = useFormik<BanGroupFormValues>({
        initialValues: {
            banType: BanType.NoComm,
            duration: Duration.dur2w,
            durationCustom: '',
            note: '',
            reason: BanReason.Cheating,
            groupId: '',
            reasonText: ''
        },
        validateOnBlur: true,
        validateOnChange: false,
        onReset: () => {
            alert('reset!');
        },
        validationSchema: validationSchema,
        onSubmit: async (values) => {
            try {
                const resp = await apiCreateBanGroup({
                    note: values.note,
                    ban_type: values.banType,
                    duration: values.duration,
                    reason: values.reason,
                    reason_text: values.reasonText,
                    target_id: values.groupId
                });
                if (!resp.status || !resp.result) {
                    sendFlash('error', 'Error saving ban');
                    return;
                }
                sendFlash('success', 'Ban created successfully');
            } catch (e) {
                logErr(e);
                sendFlash('error', 'Error saving ban');
            } finally {
                setOpen(false);
            }
        }
    });
    const formId = 'banGroupForm';
    return (
        <form onSubmit={formik.handleSubmit} id={'banForm'}>
            <Dialog
                fullWidth
                open={open}
                onClose={() => {
                    setOpen(false);
                }}
            >
                <DialogTitle component={Heading} iconLeft={<GavelIcon />}>
                    Ban Steam Group
                </DialogTitle>

                <DialogContent>
                    <Stack spacing={2}>
                        <Stack spacing={3} alignItems={'center'}>
                            <GroupIdField formik={formik} />
                            <BanTypeField formik={formik} />
                            <BanReasonField formik={formik} />
                            <BanReasonTextField formik={formik} />
                            <DurationField formik={formik} />
                            <DurationCustomField formik={formik} />
                            <NoteField formik={formik} />
                        </Stack>
                    </Stack>
                </DialogContent>
                <ModalButtons formId={formId} setOpen={setOpen} />
            </Dialog>
        </form>
    );
};
