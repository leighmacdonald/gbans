import React from 'react';
import Stack from '@mui/material/Stack';
import { apiCreateBanASN, BanReason, BanType, Duration } from '../api';
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
import { ASNumberField, ASNumberFieldValidator } from './formik/ASNumberField';

export interface BanASNModalProps {
    open: boolean;
    setOpen: (open: boolean) => void;
}

interface BanASNFormValues extends SteamIDInputValue {
    asNum: number;
    banType: BanType;
    reason: BanReason;
    reasonText: string;
    duration: Duration;
    durationCustom: string;
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
            banType: BanType.NoComm,
            duration: Duration.dur2w,
            durationCustom: '',
            note: '',
            reason: BanReason.Cheating,
            steam_id: '',
            reasonText: '',
            asNum: 0
        },
        validateOnBlur: true,
        validateOnChange: false,
        onReset: () => {
            alert('reset!');
        },
        validationSchema: validationSchema,
        onSubmit: async (values) => {
            try {
                const resp = await apiCreateBanASN({
                    note: values.note,
                    ban_type: values.banType,
                    duration: values.duration,
                    reason: values.reason,
                    reason_text: values.reasonText,
                    target_id: values.steam_id,
                    as_num: values.asNum
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
