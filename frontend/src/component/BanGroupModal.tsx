import React from 'react';
import Stack from '@mui/material/Stack';
import {
    apiCreateBanGroup,
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
import { DurationField, DurationFieldValidator } from './formik/DurationField';
import {
    DurationCustomField,
    DurationCustomFieldValidator
} from './formik/DurationCustomField';
import { NoteField, NoteFieldValidator } from './formik/NoteField';
import { ModalButtons } from './formik/ModalButtons';
import { GroupIdField, GroupIdFieldValidator } from './formik/GroupIdField';
import { SteamIdField, steamIdValidator } from './formik/SteamIdField';

export interface BanGroupModalProps
    extends ConfirmationModalProps<IAPIBanGroupRecord> {}

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
