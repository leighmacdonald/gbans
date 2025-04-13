import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import Grid from '@mui/material/Grid';
import { useForm } from '@tanstack/react-form';
import { useMutation } from '@tanstack/react-query';
import { apiCreateForumCategory, apiSaveForumCategory, ForumCategory } from '../../api/forum.ts';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';
import { Buttons } from '../field/Buttons.tsx';
import { TextFieldSimple } from '../field/TextFieldSimple.tsx';

type ForumCategoryEditorValues = {
    title: string;
    description: string;
    ordering: string;
};

// interface ForumCategoryEditorProps {
//     initial_forum_category_id?: number;
// }

// const validationSchema = yup.object({
//     title: titleFieldValidator
// });

export const ForumCategoryEditorModal = NiceModal.create(({ category }: { category?: ForumCategory }) => {
    const modal = useModal();
    const { sendError } = useUserFlashCtx();

    const mutation = useMutation({
        mutationKey: ['forumCategory'],
        mutationFn: async (values: ForumCategoryEditorValues) => {
            if (category?.forum_category_id) {
                return await apiSaveForumCategory(
                    category.forum_category_id,
                    values.title,
                    values.description,
                    Number(values.ordering)
                );
            } else {
                return await apiCreateForumCategory(values.title, values.description, Number(values.ordering));
            }
        },
        onSuccess: async (category: ForumCategory) => {
            modal.resolve(category);
            await modal.hide();
        },
        onError: sendError
    });

    const { Field, Subscribe, handleSubmit, reset } = useForm({
        onSubmit: async ({ value }) => {
            mutation.mutate({ ...value });
        },
        defaultValues: {
            title: category?.title ?? '',
            description: category?.description ?? '',
            ordering: category?.ordering ? String(category.ordering) : '1'
        }
    });

    return (
        <Dialog {...muiDialogV5(modal)} fullWidth maxWidth={'lg'}>
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await handleSubmit();
                }}
            >
                <DialogTitle>Category Editor</DialogTitle>

                <DialogContent>
                    <Grid container spacing={2}>
                        <Grid size={{ xs: 12 }}>
                            <Field
                                name={'title'}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'Title'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 12 }}>
                            <Field
                                name={'description'}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'Description'} rows={5} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 12 }}>
                            <Field
                                name={'ordering'}
                                children={(props) => {
                                    return <TextFieldSimple {...props} label={'Order'} />;
                                }}
                            />
                        </Grid>
                    </Grid>
                </DialogContent>

                <DialogActions>
                    <Grid container>
                        <Grid size={{ xs: 12 }}>
                            <Subscribe
                                selector={(state) => [state.canSubmit, state.isSubmitting]}
                                children={([canSubmit, isSubmitting]) => {
                                    return (
                                        <Buttons
                                            reset={reset}
                                            canSubmit={canSubmit}
                                            isSubmitting={isSubmitting}
                                            onClose={async () => {
                                                await modal.hide();
                                            }}
                                        />
                                    );
                                }}
                            />
                        </Grid>
                    </Grid>
                </DialogActions>
            </form>
        </Dialog>
    );
});
