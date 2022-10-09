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
import { BanTypeField } from './formik/BanTypeField';
import { BanReasonField } from './formik/BanReasonField';
import { DurationField } from './formik/DurationField';
import { DurationCustomField } from './formik/DurationCustomField';
import { NoteField } from './formik/NoteField';
import { ModalButtons } from './formik/ModalButtons';
import { GroupIdField } from './formik/GroupIdField';

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
    groupId: yup.string().min(10, 'Must be positive integer'),
    banType: yup
        .number()
        .label('Select a ban type')
        .required('ban type is required'),
    reason: yup
        .number()
        .label('Select a reason')
        .required('reason is required'),
    reasonText: yup.string().label('Custom reason'),
    duration: yup
        .string()
        .label('Ban/Mute duration')
        .required('Duration is required'),
    durationCustom: yup.string().label('Custom duration'),
    note: yup.string().label('Hidden Moderator Note')
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
