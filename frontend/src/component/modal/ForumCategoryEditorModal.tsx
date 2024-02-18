import { useCallback, useEffect } from 'react';
import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import {
    Dialog,
    DialogActions,
    DialogContent,
    DialogTitle
} from '@mui/material';
import Stack from '@mui/material/Stack';
import { Formik, useFormikContext } from 'formik';
import * as yup from 'yup';
import {
    apiCreateForumCategory,
    apiGetForumCategory,
    apiSaveForumCategory,
    ForumCategory
} from '../../api/forum';
import { logErr } from '../../util/errors';
import { titleFieldValidator } from '../../util/validators.ts';
import { DescriptionField } from '../formik/DescriptionField';
import { OrderingField } from '../formik/OrderingField';
import { TitleField } from '../formik/TitleField';
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
        const modal = useModal();

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
                initialValues={{
                    title: '',
                    description: '',
                    ordering: 0
                }}
                onSubmit={onSubmit}
                validationSchema={validationSchema}
            >
                <Dialog {...muiDialogV5(modal)} fullWidth maxWidth={'lg'}>
                    <DialogTitle>Category Editor</DialogTitle>

                    <DialogContent>
                        <CatLoader
                            initial_forum_category_id={
                                initial_forum_category_id ?? 0
                            }
                        />
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
export const CatLoader = ({
    initial_forum_category_id
}: {
    initial_forum_category_id: number;
}) => {
    const { setFieldValue } = useFormikContext<ForumCategory>();

    useEffect(() => {
        if (initial_forum_category_id) {
            apiGetForumCategory(initial_forum_category_id).then((cat) => {
                setFieldValue('title', cat.title).then(() => {
                    setFieldValue('description', cat.description).then(() => {
                        setFieldValue('ordering', cat.ordering).catch(logErr);
                    });
                });
            });
        }
    }, [initial_forum_category_id, setFieldValue]);

    return <></>;
};

export default ForumCategoryEditorModal;
