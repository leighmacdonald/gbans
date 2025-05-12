import NiceModal, { muiDialogV5, useModal } from '@ebay/nice-modal-react';
import { Dialog, DialogActions, DialogContent, DialogTitle } from '@mui/material';
import ButtonGroup from '@mui/material/ButtonGroup';
import Grid from '@mui/material/Grid';
import { useMutation } from '@tanstack/react-query';
import { z } from 'zod';
import { apiCreateForumCategory, apiSaveForumCategory, ForumCategory } from '../../api/forum.ts';
import { useAppForm } from '../../contexts/formContext.tsx';
import { useUserFlashCtx } from '../../hooks/useUserFlashCtx.ts';

const schema = z.object({
    title: z.string().min(2),
    description: z.string(),
    ordering: z.number()
});

export const ForumCategoryEditorModal = NiceModal.create(({ category }: { category?: ForumCategory }) => {
    const modal = useModal();
    const { sendError } = useUserFlashCtx();

    const mutation = useMutation({
        mutationKey: ['forumCategory'],
        mutationFn: async (values: z.input<typeof schema>) => {
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

    const form = useAppForm({
        onSubmit: async ({ value }) => {
            mutation.mutate({ ...value });
        },
        defaultValues: {
            title: category?.title ?? '',
            description: category?.description ?? '',
            ordering: category?.ordering ?? 1
        },
        validators: {
            onSubmit: schema
        }
    });

    return (
        <Dialog {...muiDialogV5(modal)} fullWidth maxWidth={'lg'}>
            <form
                onSubmit={async (e) => {
                    e.preventDefault();
                    e.stopPropagation();
                    await form.handleSubmit();
                }}
            >
                <DialogTitle>Category Editor</DialogTitle>

                <DialogContent>
                    <Grid container spacing={2}>
                        <Grid size={{ xs: 12 }}>
                            <form.AppField
                                name={'title'}
                                children={(field) => {
                                    return <field.TextField label={'Title'} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 12 }}>
                            <form.AppField
                                name={'description'}
                                children={(field) => {
                                    return <field.TextField label={'Description'} rows={5} />;
                                }}
                            />
                        </Grid>
                        <Grid size={{ xs: 12 }}>
                            <form.AppField
                                name={'ordering'}
                                children={(field) => {
                                    return <field.TextField label={'Order'} />;
                                }}
                            />
                        </Grid>
                    </Grid>
                </DialogContent>

                <DialogActions>
                    <Grid container>
                        <Grid size={{ xs: 12 }}>
                            <form.AppForm>
                                <ButtonGroup>
                                    <form.ResetButton />
                                    <form.SubmitButton />
                                </ButtonGroup>
                            </form.AppForm>
                        </Grid>
                    </Grid>
                </DialogActions>
            </form>
        </Dialog>
    );
});
