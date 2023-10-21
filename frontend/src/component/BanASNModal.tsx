import GavelIcon from '@mui/icons-material/Gavel';
import { Dialog, DialogContent, DialogTitle } from '@mui/material';
import Stack from '@mui/material/Stack';
import { useFormik } from 'formik';
import React from 'react';
import * as yup from 'yup';
import { apiCreateBanASN, BanReason, BanType, Duration } from '../api';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { logErr } from '../util/errors';
import { Heading } from './Heading';
import { ASNumberField, ASNumberFieldValidator } from './formik/ASNumberField';
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
import { NoteField, NoteFieldValidator } from './formik/NoteField';
import {
    SteamIdField,
    SteamIDInputValue,
    steamIdValidator
} from './formik/SteamIdField';

export interface BanASNModalProps {
    open: boolean;
    setOpen: (open: boolean) => void;
}

interface BanASNFormValues extends SteamIDInputValue {
    as_num: number;
    ban_type: BanType;
    reason: BanReason;
    reason_text: string;
    duration: Duration;
    duration_custom: string;
    note: string;
}

const validationSchema = yup.object({
    steam_id: steamIdValidator,
    asNum: ASNumberFieldValidator,
    banType: BanTypeFieldValidator,
    reason: BanReasonFieldValidator,
    reasonText: BanReasonTextFieldValidator,
    duration: DurationFieldValidator,
    durationCustom: DurationCustomFieldValidator,
    note: NoteFieldValidator
});

export const BanASNModal = ({ open, setOpen }: BanASNModalProps) => {
    const { sendFlash } = useUserFlashCtx();

    const formik = useFormik<BanASNFormValues>({
        initialValues: {
            ban_type: BanType.NoComm,
            duration: Duration.dur2w,
            duration_custom: '',
            note: '',
            reason: BanReason.Cheating,
            steam_id: '',
            reason_text: '',
            as_num: 0
        },
        validateOnBlur: true,
        validateOnChange: false,
        onReset: () => {
            alert('reset!');
        },
        validationSchema: validationSchema,
        onSubmit: async (values) => {
            try {
                await apiCreateBanASN({
                    note: values.note,
                    ban_type: values.ban_type,
                    duration: values.duration,
                    reason: values.reason,
                    reason_text: values.reason_text,
                    target_id: values.steam_id,
                    as_num: values.as_num
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
    const formId = 'banASNForm';

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
                        <ASNumberField formik={formik} />
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
