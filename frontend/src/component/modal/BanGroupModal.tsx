import React, { useCallback } from 'react';
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
import {
    apiCreateBanGroup,
    BanType,
    Duration,
    IAPIBanGroupRecord
} from '../../api';
import { useUserFlashCtx } from '../../contexts/UserFlashCtx';
import { logErr } from '../../util/errors';
import { Heading } from '../Heading';
import {
    DurationCustomField,
    DurationCustomFieldValidator
} from '../formik/DurationCustomField';
import { DurationField, DurationFieldValidator } from '../formik/DurationField';
import { GroupIdField, GroupIdFieldValidator } from '../formik/GroupIdField';
import { NoteField, NoteFieldValidator } from '../formik/NoteField';
import { SteamIdField, steamIdValidator } from '../formik/SteamIdField';
import { CancelButton, ResetButton, SaveButton } from './Buttons';
import { ConfirmationModalProps } from './ConfirmationModal';

export interface BanGroupModalProps
    extends ConfirmationModalProps<IAPIBanGroupRecord> {
    asnNum?: number;
}

export interface BanGroupFormValues {
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

export const BanGroupModal = NiceModal.create(() => {
    const { sendFlash } = useUserFlashCtx();
    const modal = useModal();
    const onSubmit = useCallback(
        async (values: BanGroupFormValues) => {
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
                await modal.hide();
            }
        },
        [modal, sendFlash]
    );

    const formId = 'banGroupForm';

    return (
        <Formik
            onSubmit={onSubmit}
            id={formId}
            initialValues={{
                steam_id: '',
                duration: Duration.dur2w,
                duration_custom: '',
                note: '',
                group_id: ''
            }}
            validateOnBlur={true}
            validateOnChange={false}
            validationSchema={validationSchema}
        >
            <Dialog fullWidth {...muiDialogV5(modal)}>
                <DialogTitle component={Heading} iconLeft={<GavelIcon />}>
                    Ban Steam Group
                </DialogTitle>

                <DialogContent>
                    <Stack spacing={2}>
                        <Stack spacing={3} alignItems={'center'}>
                            <SteamIdField fullWidth />
                            <GroupIdField />
                            <DurationField />
                            <DurationCustomField<BanGroupFormValues> />
                            <NoteField />
                        </Stack>
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
});
