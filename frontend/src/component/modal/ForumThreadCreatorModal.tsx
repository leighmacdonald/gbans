import { useCallback } from 'react';
import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import {
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle
} from '@mui/material';
import Stack from '@mui/material/Stack';
import { useTheme } from '@mui/material/styles';
import useMediaQuery from '@mui/material/useMediaQuery';
import { Formik } from 'formik';
import * as yup from 'yup';
import { apiCreateThread } from '../../api/forum';
import { logErr } from '../../util/errors';
import { bodyMDValidator, titleFieldValidator } from '../../util/validators.ts';
import { MDBodyField } from '../MDBodyField.tsx';
import { LockedField } from '../formik/LockedField';
import { StickyField } from '../formik/StickyField';
import { TitleField } from '../formik/TitleField';
import { CancelButton, SubmitButton } from './Buttons';
import { ModalConfirm, ModalForumThreadCreator } from './index';

interface ForumThreadEditorValues {
    forum_id: number;
    title: string;
    body_md: string;
    sticky: boolean;
    locked: boolean;
}

interface ForumThreadEditorProps {
    forum_id: number;
}

const validationSchema = yup.object({
    title: titleFieldValidator,
    body_md: bodyMDValidator
});

export const ForumThreadCreatorModal = NiceModal.create(
    ({ forum_id }: ForumThreadEditorProps) => {
        const threadModal = useModal(ModalForumThreadCreator);
        const confirmModal = useModal(ModalConfirm);
        const theme = useTheme();
        const fullScreen = useMediaQuery(theme.breakpoints.down('md'));

        const iv: ForumThreadEditorValues = {
            forum_id: forum_id,
            body_md: '',
            title: '',
            locked: false,
            sticky: false
        };

        const onSubmit = useCallback(
            async (values: ForumThreadEditorValues) => {
                try {
                    threadModal.resolve(
                        await apiCreateThread(
                            forum_id,
                            values.title,
                            values.body_md,
                            values.sticky,
                            values.locked
                        )
                    );

                    await threadModal.hide();
                } catch (e) {
                    threadModal.reject(e);
                }
            },
            [forum_id, threadModal]
        );

        const onClose = useCallback(
            async (_: unknown, reason: 'escapeKeyDown' | 'backdropClick') => {
                if (reason == 'backdropClick') {
                    try {
                        const confirmed = await confirmModal.show({
                            title: 'Cancel thread creation?',
                            children: 'All progress will be lost'
                        });
                        if (confirmed) {
                            await confirmModal.hide();
                            await threadModal.hide();
                        } else {
                            await confirmModal.hide();
                        }
                    } catch (e) {
                        logErr(e);
                    }
                }
            },
            [confirmModal, threadModal]
        );

        return (
            <Formik<ForumThreadEditorValues>
                initialValues={iv}
                onSubmit={onSubmit}
                validationSchema={validationSchema}
            >
                <Dialog
                    {...muiDialogV5(threadModal)}
                    fullWidth
                    maxWidth={'lg'}
                    closeAfterTransition={false}
                    onClose={onClose}
                    fullScreen={fullScreen}
                >
                    <DialogTitle>Create New Thread</DialogTitle>
                    <DialogContent>
                        <Stack spacing={2}>
                            <TitleField />
                            <MDBodyField />
                            <StickyField />
                            <LockedField />
                        </Stack>
                    </DialogContent>
                    <DialogActions>
                        <CancelButton />
                        <SubmitButton label={'Post'} />
                    </DialogActions>
                </Dialog>
            </Formik>
        );
    }
);

export default ForumThreadCreatorModal;
