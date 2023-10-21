import React from 'react';
import GavelIcon from '@mui/icons-material/Gavel';
import { Dialog, DialogContent, DialogTitle } from '@mui/material';
import Stack from '@mui/material/Stack';
import { useFormik } from 'formik';
import * as yup from 'yup';
import { apiCreateBanCIDR, BanReason, BanType, Duration } from '../api';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { logErr } from '../util/errors';
import { Heading } from './Heading';
import {
    BanReasonField,
    BanReasonFieldValidator
} from './formik/BanReasonField';
import {
    BanReasonTextField,
    BanReasonTextFieldValidator
} from './formik/BanReasonTextField';
import { BanTypeField, BanTypeFieldValidator } from './formik/BanTypeField';
import {
    DurationCustomField,
    DurationCustomFieldValidator
} from './formik/DurationCustomField';
import { DurationField, DurationFieldValidator } from './formik/DurationField';
import { ModalButtons } from './formik/ModalButtons';
import {
    NetworkRangeField,
    NetworkRangeFieldValidator
} from './formik/NetworkRangeField';
import { NoteField, NoteFieldValidator } from './formik/NoteField';
import {
    SteamIdField,
    SteamIDInputValue,
    steamIdValidator
} from './formik/SteamIdField';

export interface BanCIDRModalProps {
    open: boolean;
    setOpen: (open: boolean) => void;
}

interface BanCIDRFormValues extends SteamIDInputValue {
    cidr: string;
    ban_type: BanType;
    reason: BanReason;
    reason_text: string;
    duration: Duration;
    duration_custom: string;
    note: string;
}

const validationSchema = yup.object({
    steam_id: steamIdValidator,
    cidr: NetworkRangeFieldValidator,
    banType: BanTypeFieldValidator,
    reason: BanReasonFieldValidator,
    reasonText: BanReasonTextFieldValidator,
    duration: DurationFieldValidator,
    durationCustom: DurationCustomFieldValidator,
    note: NoteFieldValidator
});

export const BanCIDRModal = ({ open, setOpen }: BanCIDRModalProps) => {
    const { sendFlash } = useUserFlashCtx();

    const formik = useFormik<BanCIDRFormValues>({
        initialValues: {
            ban_type: BanType.NoComm,
            duration: Duration.dur2w,
            duration_custom: '',
            note: '',
            reason: BanReason.Cheating,
            steam_id: '',
            reason_text: '',
            cidr: ''
        },
        validateOnBlur: true,
        validateOnChange: false,
        onReset: () => {
            alert('reset!');
        },
        validationSchema: validationSchema,
        onSubmit: async (values) => {
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
                setOpen(false);
            }
        }
    });

    const formId = 'banCIDRForm';

    return (
        <form onSubmit={formik.handleSubmit} id={formId}>
            <Dialog
                fullWidth
                open={open}
                onClose={() => {
                    setOpen(false);
                }}
            >
                <DialogTitle component={Heading} iconLeft={<GavelIcon />}>
                    Ban Steam Profile
                </DialogTitle>

                <DialogContent>
                    <Stack spacing={2}>
                        <SteamIdField
                            formik={formik}
                            fullWidth
                            isReadOnly={false}
                        />
                        <SteamIdField
                            formik={formik}
                            fullWidth
                            isReadOnly={false}
                        />
                        <NetworkRangeField formik={formik} />
                        <BanTypeField formik={formik} />
                        <BanReasonField formik={formik} />
                        <BanReasonTextField formik={formik} />
                        <DurationField formik={formik} />
                        <DurationCustomField formik={formik} />
                        <NoteField formik={formik} />
                    </Stack>
                </DialogContent>
                <ModalButtons
                    formId={formId}
                    setOpen={setOpen}
                    inProgress={false}
                />
            </Dialog>
        </form>
    );
};
