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
import {
    apiCreateForumCategory,
    apiGetForumCategory,
    apiSaveForumCategory,
    ForumCategory
} from '../../api/forum';
import { DescriptionField } from '../formik/DescriptionField';
import { OrderingField } from '../formik/OrderingField';
import { TitleField, titleFieldValidator } from '../formik/TitleField';
import { CancelButton, SubmitButton } from './Buttons';

interface ForumCategoryEditorValues {
    title: string;
    description: string;
    ordering: number;
}

interface ForumCategoryEditorProps {
    initial_forum_category_id?: number;
}

const validationSchema = yup.object({
    title: titleFieldValidator
});

export const ForumCategoryEditorModal = NiceModal.create(
    ({ initial_forum_category_id }: ForumCategoryEditorProps) => {
        const [category, setCategory] = useState<ForumCategory>();
        const modal = useModal();

        useEffect(() => {
            if (initial_forum_category_id) {
                apiGetForumCategory(initial_forum_category_id).then((cat) => {
                    setCategory(cat);
                });
            }
        }, [initial_forum_category_id]);

        const iv: ForumCategoryEditorValues = {
            title: category?.title ?? '',
            description: category?.description ?? '',
            ordering: category?.ordering ?? 0
        };

        const onSubmit = useCallback(
            async (values: ForumCategoryEditorValues) => {
                try {
                    if (initial_forum_category_id) {
                        modal.resolve(
                            await apiSaveForumCategory(
                                initial_forum_category_id,
                                values.title,
                                values.description,
                                values.ordering
                            )
                        );
                    } else {
                        modal.resolve(
                            await apiCreateForumCategory(
                                values.title,
                                values.description,
                                values.ordering
                            )
                        );
                    }
                } catch (e) {
                    modal.reject(e);
                } finally {
                    await modal.hide();
                }
            },
            [initial_forum_category_id, modal]
        );

        return (
            <Formik<ForumCategoryEditorValues>
                initialValues={iv}
                onSubmit={onSubmit}
                validationSchema={validationSchema}
            >
                <Dialog {...muiDialogV5(modal)}>
                    <DialogTitle>Category Editor</DialogTitle>

                    <DialogContent>
                        <Stack spacing={2}>
                            <TitleField />
                            <DescriptionField />
                            <OrderingField />
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
