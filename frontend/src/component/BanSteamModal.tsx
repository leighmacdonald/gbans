import React, { useMemo } from 'react';
import Stack from '@mui/material/Stack';
import { apiCreateBanSteam, BanReason, BanType, Duration } from '../api';
import { Heading } from './Heading';
import SteamID from 'steamid';
import * as yup from 'yup';
import { useFormik } from 'formik';
import GavelIcon from '@mui/icons-material/Gavel';
import { Dialog, DialogContent, DialogTitle } from '@mui/material';
import {
    SteamIdField,
    SteamIDInputValue,
    steamIdValidator
} from './formik/SteamIdField';
import { logErr } from '../util/errors';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { BanTypeField } from './formik/BanTypeField';
import { BanReasonField } from './formik/BanReasonField';
import { DurationField } from './formik/DurationField';
import { DurationCustomField } from './formik/DurationCustomField';
import { NoteField } from './formik/NoteField';
import { ReportIdField } from './formik/ReportIdField';
import { ModalButtons } from './formik/ModalButtons';
import { BanReasonTextField } from './formik/BanReasonTextField';

export interface BanModalProps {
    open: boolean;
    setOpen: (open: boolean) => void;
    reportId?: number;
    steamId?: SteamID;
}

interface BanSteamFormValues extends SteamIDInputValue {
    reportId?: number;
    banType: BanType;
    reason: BanReason;
    reasonText: string;
    duration: Duration;
    durationCustom: string;
    note: string;
}

const validationSchema = yup.object({
    steam_id: steamIdValidator,
    reportId: yup.number().min(0, 'Must be positive integer').nullable(),
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

export const BanSteamModal = ({
    open,
    setOpen,
    steamId,
    reportId
}: BanModalProps) => {
    const { sendFlash } = useUserFlashCtx();

    const isReadOnlySid = useMemo(() => {
        return !!steamId?.getSteamID64();
    }, [steamId]);

    const formik = useFormik<BanSteamFormValues>({
        initialValues: {
            banType: BanType.NoComm,
            duration: Duration.dur2w,
            durationCustom: '',
            note: '',
            reason: BanReason.Cheating,
            steam_id: steamId?.getSteamID64() ?? '',
            reasonText: '',
            reportId: reportId
        },
        validateOnBlur: true,
        validateOnChange: false,
        onReset: () => {
            alert('reset!');
        },
        validationSchema: validationSchema,
        onSubmit: async (values) => {
            try {
                const resp = await apiCreateBanSteam({
                    note: values.note,
                    ban_type: values.banType,
                    duration: values.duration,
                    reason: values.reason,
                    reason_text: values.reasonText,
                    report_id: values.reportId,
                    target_id: values.steam_id
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
    const formId = 'banSteamForm';
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
                            isReadOnly={isReadOnlySid}
                        />
                        <ReportIdField formik={formik} />
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
