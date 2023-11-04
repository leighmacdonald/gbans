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
    BanReason,
    BanType,
    Duration,
    IAPIBanGroupRecord
} from '../../api';
import { Heading } from '../Heading';
import { BanReasonField } from '../formik/BanReasonField';
import { BanReasonTextField } from '../formik/BanReasonTextField';
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
    reason: BanReason;
    reason_text: string;
}

export const validationSchema = yup.object({
    steam_id: steamIdValidator,
    groupId: GroupIdFieldValidator,
    duration: DurationFieldValidator,
    durationCustom: DurationCustomFieldValidator,
    note: NoteFieldValidator
});

export const BanGroupModal = NiceModal.create(() => {
    const modal = useModal();
    const onSubmit = useCallback(
        async (values: BanGroupFormValues) => {
            try {
                const record = await apiCreateBanGroup({
                    group_id: values.group_id,
                    note: values.note,
                    ban_type: BanType.Banned,
                    duration: values.duration,
                    target_id: values.steam_id,
                    reason: values.reason,
                    reason_text: values.reason_text
                });
                modal.resolve(record);
            } catch (e) {
                modal.reject(e);
            } finally {
                await modal.hide();
            }
        },
        [modal]
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
                reason: BanReason.Cheating,
                reason_text: '',
                note: '',
                group_id: ''
            }}
            validateOnBlur={true}
            validateOnChange={false}
            //validationSchema={validationSchema}
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
                            <BanReasonField />
                            <BanReasonTextField />
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
