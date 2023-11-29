import React, { useCallback, useEffect, useState } from 'react';
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
import {
    apiCreateThread,
    apiGetThread,
    apiSaveThread,
    ForumThread
} from '../../api/forum';
import { logErr } from '../../util/errors';
import { BodyMDField, bodyMDValidator } from '../formik/BodyMDField';
import { LockedField } from '../formik/LockedField';
import { StickyField } from '../formik/StickyField';
import { TitleField, titleFieldValidator } from '../formik/TitleField';
import { CancelButton, SubmitButton } from './Buttons';
import { ModalConfirm, ModalForumThreadEditor } from './index';

interface ForumThreadEditorValues {
    forum_id: number;
    title: string;
    body_md: string;
    sticky: boolean;
    locked: boolean;
}

interface ForumThreadEditorProps {
    forum_id: number;
    thread_id?: number;
}

const validationSchema = yup.object({
    title: titleFieldValidator,
    body_md: bodyMDValidator
});

export const ForumThreadEditorModal = NiceModal.create(
    ({ forum_id, thread_id }: ForumThreadEditorProps) => {
        const [thread, setThread] = useState<ForumThread>();
        const threadModal = useModal(ModalForumThreadEditor);
        const confirmModal = useModal(ModalConfirm);
        const theme = useTheme();
        const fullScreen = useMediaQuery(theme.breakpoints.down('md'));

        useEffect(() => {
            const abortController = new AbortController();
            if (thread_id && thread_id > 0) {
                apiGetThread(thread_id)
                    .then((t) => setThread(t))
                    .catch(logErr);
            }
            return () => abortController.abort();
        }, [thread_id]);

        const iv: ForumThreadEditorValues = {
            forum_id: forum_id,
            body_md: thread?.message?.body_md ?? '',
            title: thread?.title ?? '',
            locked: thread?.locked ?? false,
            sticky: thread?.sticky ?? false
        };

        const onSubmit = useCallback(
            async (values: ForumThreadEditorValues) => {
                try {
                    if (thread_id != undefined) {
                        threadModal.resolve(
                            await apiSaveThread(
                                thread_id,
                                values.title,
                                values.body_md,
                                values.sticky,
                                values.locked
                            )
                        );
                    } else {
                        threadModal.resolve(
                            await apiCreateThread(
                                forum_id,
                                values.title,
                                values.body_md,
                                values.sticky,
                                values.locked
                            )
                        );
                    }
                    await threadModal.hide();
                } catch (e) {
                    threadModal.reject(e);
                }
            },
            [forum_id, threadModal, thread_id]
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
                            <BodyMDField />
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
