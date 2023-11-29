import React, { useCallback, useEffect, useState } from 'react';
import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import {
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle
} from '@mui/material';
import Stack from '@mui/material/Stack';
import { Formik } from 'formik';
import * as yup from 'yup';
import { PermissionLevel } from '../../api';
import { apiCreateForum, apiForum, apiSaveForum, Forum } from '../../api/forum';
import { DescriptionField } from '../formik/DescriptionField';
import { ForumCategorySelectField } from '../formik/ForumCategorySelectField';
import { OrderingField } from '../formik/OrderingField';
import { PermissionLevelField } from '../formik/PermissionLevelField';
import { TitleField, titleFieldValidator } from '../formik/TitleField';
import { CancelButton, SubmitButton } from './Buttons';

interface ForumEditorValues {
    forum_category_id: number;
    title: string;
    description: string;
    ordering: number;
    permission_level: PermissionLevel;
}

interface ForumEditorProps {
    initial_forum_id?: number;
}

const validationSchema = yup.object({
    title: titleFieldValidator
});

export const ForumForumEditorModal = NiceModal.create(
    ({ initial_forum_id }: ForumEditorProps) => {
        const [forum, setForum] = useState<Forum>();
        const modal = useModal();

        useEffect(() => {
            if (initial_forum_id) {
                apiForum(initial_forum_id).then((f) => {
                    setForum(f);
                });
            }
        }, [initial_forum_id]);

        const iv: ForumEditorValues = {
            forum_category_id: forum?.forum_category_id ?? 0,
            title: forum?.title ?? '',
            description: forum?.description ?? '',
            ordering: forum?.ordering ?? 0,
            permission_level: forum?.permission_level ?? PermissionLevel.Guest
        };

        const onSubmit = useCallback(
            async (values: ForumEditorValues) => {
                try {
                    if (initial_forum_id) {
                        modal.resolve(
                            await apiSaveForum(
                                initial_forum_id,
                                values.forum_category_id,
                                values.title,
                                values.description,
                                values.ordering,
                                values.permission_level
                            )
                        );
                    } else {
                        modal.resolve(
                            await apiCreateForum(
                                values.forum_category_id,
                                values.title,
                                values.description,
                                values.ordering,
                                values.permission_level
                            )
                        );
                    }
                } catch (e) {
                    modal.reject(e);
                } finally {
                    await modal.hide();
                }
            },
            [initial_forum_id, modal]
        );

        return (
            <Formik<ForumEditorValues>
                initialValues={iv}
                onSubmit={onSubmit}
                validationSchema={validationSchema}
            >
                <Dialog {...muiDialogV5(modal)} fullWidth maxWidth={'lg'}>
                    <DialogTitle>Category Editor</DialogTitle>

                    <DialogContent>
                        <Stack spacing={2}>
                            <ForumCategorySelectField />
                            <TitleField />
                            <DescriptionField />
                            <OrderingField />
                            <PermissionLevelField
                                levels={[
                                    PermissionLevel.Guest,
                                    PermissionLevel.User,
                                    PermissionLevel.Reserved,
                                    PermissionLevel.Editor,
                                    PermissionLevel.Moderator,
                                    PermissionLevel.Admin
                                ]}
                            />
                        </Stack>
                    </DialogContent>

                    <DialogActions>
                        <CancelButton />
                        <SubmitButton />
                    </DialogActions>
                </Dialog>
            </Formik>
        );
    }
);