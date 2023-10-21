import GavelIcon from '@mui/icons-material/Gavel';
import { Dialog, DialogContent, DialogTitle } from '@mui/material';
import Stack from '@mui/material/Stack';
import { useFormik } from 'formik';
import React from 'react';
import * as yup from 'yup';
import {
    apiCreateBanGroup,
    BanType,
    Duration,
    IAPIBanGroupRecord
} from '../api';
import { useUserFlashCtx } from '../contexts/UserFlashCtx';
import { logErr } from '../util/errors';
import { ConfirmationModalProps } from './ConfirmationModal';
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
import { GroupIdField, GroupIdFieldValidator } from './formik/GroupIdField';
import { ModalButtons } from './formik/ModalButtons';
import { NoteField, NoteFieldValidator } from './formik/NoteField';
import { GroupIdField, GroupIdFieldValidator } from './formik/GroupIdField';
import { SteamIdField, steamIdValidator } from './formik/SteamIdField';

export interface BanGroupModalProps
    extends ConfirmationModalProps<IAPIBanGroupRecord> {
    asnNum?: number;
}

interface BanGroupFormValues {
    steam_id: string;
    group_id: string;
    duration: Duration;
    duration_custom: string;
    note: string;
}

const validationSchema = yup.object({
    steam_id: steamIdValidator,
    groupId: GroupIdFieldValidator,
    duration: DurationFieldValidator,
    durationCustom: DurationCustomFieldValidator,
    note: NoteFieldValidator
});

export const BanGroupModal = ({ open, setOpen }: BanGroupModalProps) => {
    const { sendFlash } = useUserFlashCtx();

    const formik = useFormik<BanGroupFormValues>({
        initialValues: {
            steam_id: '',
            duration: Duration.dur2w,
            duration_custom: '',
            note: '',
            group_id: ''
        },
        validateOnBlur: true,
        validateOnChange: false,
        validationSchema: validationSchema,
        onSubmit: async (values) => {
            console.log('fire');
            try {
                await apiCreateBanGroup({
                    group_id: values.group_id,
                    note: values.note,
                    ban_type: BanType.Banned,
                    duration: values.duration,
                    target_id: values.steam_id
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

    const formId = 'banGroupForm';

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
                    Ban Steam Group
                </DialogTitle>

                <DialogContent>
                    <Stack spacing={2}>
                        <Stack spacing={3} alignItems={'center'}>
                            <SteamIdField fullWidth={true} formik={formik} />
                            <GroupIdField formik={formik} />
                            <DurationField formik={formik} />
                            <DurationCustomField formik={formik} />
                            <NoteField formik={formik} />
                        </Stack>
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
