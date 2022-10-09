import React from 'react';
import Stack from '@mui/material/Stack';
import { apiCreateBanCIDR, BanReason, BanType, Duration } from '../api';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { Heading } from './Heading';
import { logErr } from '../util/errors';
import {
    SteamIdField,
    SteamIDInputValue,
    steamIdValidator
} from './formik/SteamIdField';
import * as yup from 'yup';
import { BanTypeField, BanTypeFieldValidator } from './formik/BanTypeField';
import {
    BanReasonField,
    BanReasonFieldValidator
} from './formik/BanReasonField';
import {
    BanReasonTextField,
    BanReasonTextFieldValidator
} from './formik/BanReasonTextField';
import { DurationField, DurationFieldValidator } from './formik/DurationField';
import {
    DurationCustomField,
    DurationCustomFieldValidator
} from './formik/DurationCustomField';
import { NoteField, NoteFieldValidator } from './formik/NoteField';
import { Dialog, DialogContent, DialogTitle } from '@mui/material';
import GavelIcon from '@mui/icons-material/Gavel';
import { ModalButtons } from './formik/ModalButtons';
import { useFormik } from 'formik';
import {
    NetworkRangeField,
    NetworkRangeFieldValidator
} from './formik/NetworkRangeField';

export interface BanCIDRModalProps {
    open: boolean;
    setOpen: (open: boolean) => void;
}

interface BanCIDRFormValues extends SteamIDInputValue {
    cidr: string;
    banType: BanType;
    reason: BanReason;
    reasonText: string;
    duration: Duration;
    durationCustom: string;
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
            banType: BanType.NoComm,
            duration: Duration.dur2w,
            durationCustom: '',
            note: '',
            reason: BanReason.Cheating,
            steam_id: '',
            reasonText: '',
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
                const resp = await apiCreateBanCIDR({
                    note: values.note,
                    ban_type: values.banType,
                    duration: values.duration,
                    reason: values.reason,
                    reason_text: values.reasonText,
                    target_id: values.steam_id,
                    cidr: values.cidr
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
                        <NetworkRangeField formik={formik} />
                        <BanTypeField formik={formik} />
                        <BanReasonField formik={formik} />
                        <BanReasonTextField formik={formik} />
                        <DurationField formik={formik} />
                        <DurationCustomField formik={formik} />
                        <NoteField formik={formik} />
                    </Stack>
                </DialogContent>
                <ModalButtons formId={formId} setOpen={setOpen} />
            </Dialog>
        </form>
    );
};
